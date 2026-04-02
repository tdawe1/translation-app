import json
import os
import socket
import subprocess
import time
from decimal import Decimal
from pathlib import Path
from urllib.parse import urlparse

import httpx
import psycopg
from emergentintegrations.payments.stripe.checkout import (
    CheckoutSessionRequest,
    StripeCheckout,
)
from fastapi import FastAPI, Request, Response
from fastapi.responses import JSONResponse
from pydantic import BaseModel


APP_DIR = Path(__file__).resolve().parent
GO_BINARY = Path("/tmp/gengowatcher-server")
GO_LOG = Path("/tmp/gengowatcher-go.log")
GO_PORT = 8002
GO_PROCESS: subprocess.Popen | None = None
HTTP_CLIENT: httpx.AsyncClient | None = None

DEFAULT_ENV = {
    "ENV": "development",
    "PORT": str(GO_PORT),
    "DB_HOST": "127.0.0.1",
    "DB_PORT": "5432",
    "DB_USER": "gengo",
    "DB_PASSWORD": "devpass",
    "DB_NAME": "gengowatcher",
    "DB_SSLMODE": "disable",
    "JWT_SECRET": "launch-ready-development-secret-key-with-32-chars-min",
    "REDIS_URL": "redis://127.0.0.1:6379/0",
    "FRONTEND_URL": "http://127.0.0.1:3000",
    "OAUTH_REDIRECT_URL": "http://127.0.0.1:8002",
    "ALLOWED_ORIGINS": "http://127.0.0.1:3000,http://localhost:3000",
    "COOKIE_SAMESITE": "Lax",
}

PLANS = {
    "pro": {
        "id": "pro",
        "name": "Pro",
        "amount": 29.00,
        "currency": "usd",
        "description": "For individual translators who need live watcher alerts and review tools.",
        "features": [
            "Realtime job watcher",
            "Translation review workspace",
            "Priority email support",
            "3 team members included",
        ],
    },
    "team": {
        "id": "team",
        "name": "Team",
        "amount": 79.00,
        "currency": "usd",
        "description": "For fast-moving teams coordinating multiple translators and reviewers.",
        "features": [
            "Everything in Pro",
            "Unlimited watcher presets",
            "Shared review visibility",
            "Priority launch concierge",
        ],
    },
}


class CheckoutRequestBody(BaseModel):
    plan_id: str
    origin_url: str
    user_email: str | None = None


def get_db_connection() -> psycopg.Connection:
    env = merged_env()
    return psycopg.connect(
        host=env["DB_HOST"],
        port=env["DB_PORT"],
        dbname=env["DB_NAME"],
        user=env["DB_USER"],
        password=env["DB_PASSWORD"],
        autocommit=True,
    )


def ensure_payment_transactions() -> None:
    query = """
    CREATE TABLE IF NOT EXISTS payment_transactions (
        id BIGSERIAL PRIMARY KEY,
        session_id TEXT UNIQUE NOT NULL,
        plan_id TEXT NOT NULL,
        plan_name TEXT NOT NULL,
        amount NUMERIC(10, 2) NOT NULL,
        currency TEXT NOT NULL,
        status TEXT NOT NULL,
        payment_status TEXT NOT NULL,
        user_email TEXT,
        metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
        checkout_url TEXT,
        created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
        processed_at TIMESTAMPTZ
    );
    """
    with get_db_connection() as conn:
        with conn.cursor() as cur:
            cur.execute(query)


def upsert_transaction(
    *,
    session_id: str,
    plan_id: str,
    plan_name: str,
    amount: float,
    currency: str,
    status: str,
    payment_status: str,
    user_email: str | None,
    metadata: dict[str, str],
    checkout_url: str | None,
) -> None:
    query = """
    INSERT INTO payment_transactions (
        session_id, plan_id, plan_name, amount, currency, status, payment_status, user_email, metadata, checkout_url
    ) VALUES (%s, %s, %s, %s, %s, %s, %s, %s, %s::jsonb, %s)
    ON CONFLICT (session_id) DO UPDATE SET
        status = EXCLUDED.status,
        payment_status = EXCLUDED.payment_status,
        metadata = EXCLUDED.metadata,
        checkout_url = COALESCE(EXCLUDED.checkout_url, payment_transactions.checkout_url),
        updated_at = NOW(),
        processed_at = CASE
            WHEN EXCLUDED.payment_status = 'paid' THEN NOW()
            ELSE payment_transactions.processed_at
        END;
    """
    with get_db_connection() as conn:
        with conn.cursor() as cur:
            cur.execute(
                query,
                (
                    session_id,
                    plan_id,
                    plan_name,
                    Decimal(f"{amount:.2f}"),
                    currency,
                    status,
                    payment_status,
                    user_email,
                    json.dumps(metadata),
                    checkout_url,
                ),
            )


def get_transaction(session_id: str) -> dict | None:
    query = """
    SELECT session_id, plan_id, plan_name, amount::float8, currency, status, payment_status, user_email, metadata,
           created_at, updated_at, processed_at
    FROM payment_transactions
    WHERE session_id = %s;
    """
    with get_db_connection() as conn:
        with conn.cursor(row_factory=psycopg.rows.dict_row) as cur:
            cur.execute(query, (session_id,))
            row = cur.fetchone()
    return dict(row) if row else None


def normalize_origin(origin_url: str, request: Request) -> str:
    parsed = urlparse(origin_url)
    if parsed.scheme in {"http", "https"} and parsed.netloc:
        return origin_url.rstrip("/")
    return str(request.base_url).rstrip("/")


def get_stripe_checkout(request: Request) -> StripeCheckout:
    api_key = os.environ.get("STRIPE_API_KEY", "")
    host_url = str(request.base_url).rstrip("/")
    webhook_url = f"{host_url}/api/webhook/stripe"
    return StripeCheckout(api_key=api_key, webhook_url=webhook_url)


def load_env_file() -> None:
    env_path = APP_DIR / ".env"
    if not env_path.exists():
        return

    for raw_line in env_path.read_text().splitlines():
        line = raw_line.strip()
        if not line or line.startswith("#") or "=" not in line:
            continue
        key, value = line.split("=", 1)
        os.environ.setdefault(key.strip(), value.strip())


def merged_env() -> dict[str, str]:
    env = os.environ.copy()
    for key, value in DEFAULT_ENV.items():
        env.setdefault(key, value)
    env["PORT"] = str(GO_PORT)
    env["PATH"] = f"/usr/local/go/bin:{env.get('PATH', '')}"
    env.setdefault("HOME", "/root")
    env.setdefault("GOPATH", "/tmp/go")
    env.setdefault("GOMODCACHE", "/tmp/go/pkg/mod")
    env.setdefault("GOCACHE", "/tmp/go-build")
    return env


def port_open(host: str, port: int) -> bool:
    with socket.socket(socket.AF_INET, socket.SOCK_STREAM) as sock:
        sock.settimeout(1)
        return sock.connect_ex((host, port)) == 0


def ensure_postgres() -> None:
    if subprocess.run(["pg_isready", "-q"], check=False).returncode == 0:
        return

    subprocess.run(["pg_ctlcluster", "15", "main", "start"], check=False)
    for _ in range(30):
        if subprocess.run(["pg_isready", "-q"], check=False).returncode == 0:
            return
        time.sleep(1)
    raise RuntimeError("PostgreSQL did not start in time")


def ensure_database() -> None:
    sql = """
DO $$
BEGIN
  IF NOT EXISTS (SELECT FROM pg_catalog.pg_roles WHERE rolname = 'gengo') THEN
    CREATE ROLE gengo LOGIN PASSWORD 'devpass';
  END IF;
END
$$;
"""
    subprocess.run(
        ["runuser", "-u", "postgres", "--", "psql", "-v", "ON_ERROR_STOP=1", "-c", sql],
        check=True,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )

    db_exists = subprocess.run(
        [
            "runuser",
            "-u",
            "postgres",
            "--",
            "psql",
            "-tAc",
            "SELECT 1 FROM pg_database WHERE datname='gengowatcher';",
        ],
        check=True,
        capture_output=True,
        text=True,
    ).stdout.strip()

    if db_exists != "1":
        subprocess.run(
            ["runuser", "-u", "postgres", "--", "createdb", "-O", "gengo", "gengowatcher"],
            check=True,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )


def ensure_redis() -> None:
    ping = subprocess.run(["redis-cli", "ping"], check=False, capture_output=True, text=True)
    if ping.returncode == 0 and ping.stdout.strip() == "PONG":
        return

    subprocess.run(["redis-server", "--daemonize", "yes"], check=True)
    for _ in range(15):
        ping = subprocess.run(["redis-cli", "ping"], check=False, capture_output=True, text=True)
        if ping.returncode == 0 and ping.stdout.strip() == "PONG":
            return
        time.sleep(1)
    raise RuntimeError("Redis did not start in time")


def latest_source_mtime() -> float:
    patterns = ["cmd/**/*.go", "internal/**/*.go", "go.mod", "go.sum"]
    mtimes: list[float] = []
    for pattern in patterns:
        for path in APP_DIR.glob(pattern):
            if path.is_file():
                mtimes.append(path.stat().st_mtime)
    return max(mtimes) if mtimes else 0


def ensure_go_binary() -> None:
    GO_BINARY.parent.mkdir(parents=True, exist_ok=True)
    if GO_BINARY.exists() and GO_BINARY.stat().st_mtime >= latest_source_mtime():
        return

    subprocess.run(
        ["/usr/local/go/bin/go", "build", "-o", str(GO_BINARY), "./cmd/server"],
        cwd=APP_DIR,
        env=merged_env(),
        check=True,
    )


def read_go_log() -> str:
    if not GO_LOG.exists():
        return ""
    lines = GO_LOG.read_text().splitlines()
    return "\n".join(lines[-30:])


def ensure_go_process() -> None:
    global GO_PROCESS

    if GO_PROCESS and GO_PROCESS.poll() is None and port_open("127.0.0.1", GO_PORT):
        return

    ensure_go_binary()
    GO_LOG.parent.mkdir(parents=True, exist_ok=True)
    log_handle = GO_LOG.open("a")
    GO_PROCESS = subprocess.Popen(
        [str(GO_BINARY)],
        cwd=APP_DIR,
        env=merged_env(),
        stdout=log_handle,
        stderr=log_handle,
        start_new_session=True,
    )

    for _ in range(40):
        if GO_PROCESS.poll() is not None:
            break
        if port_open("127.0.0.1", GO_PORT):
            return
        time.sleep(1)

    raise RuntimeError(f"Go backend failed to start.\n{read_go_log()}")


def stop_go_process() -> None:
    global GO_PROCESS
    if GO_PROCESS and GO_PROCESS.poll() is None:
        GO_PROCESS.terminate()
        try:
            GO_PROCESS.wait(timeout=10)
        except subprocess.TimeoutExpired:
            GO_PROCESS.kill()
    GO_PROCESS = None


async def proxy_request(request: Request, path: str) -> Response:
    global HTTP_CLIENT
    if HTTP_CLIENT is None:
        HTTP_CLIENT = httpx.AsyncClient(timeout=httpx.Timeout(60.0, connect=10.0), follow_redirects=False)

    target = f"http://127.0.0.1:{GO_PORT}/{path}"
    if request.url.query:
        target = f"{target}?{request.url.query}"

    body = await request.body()
    headers = {
        key: value
        for key, value in request.headers.items()
        if key.lower() not in {"host", "content-length", "connection"}
    }

    upstream = await HTTP_CLIENT.request(
        request.method,
        target,
        content=body,
        headers=headers,
    )

    excluded_headers = {"content-encoding", "transfer-encoding", "connection"}
    response_headers = {
        key: value for key, value in upstream.headers.items() if key.lower() not in excluded_headers
    }
    return Response(content=upstream.content, status_code=upstream.status_code, headers=response_headers)


load_env_file()
app = FastAPI()


@app.on_event("startup")
async def startup_event() -> None:
    ensure_postgres()
    ensure_database()
    ensure_payment_transactions()
    ensure_redis()
    ensure_go_process()


@app.on_event("shutdown")
async def shutdown_event() -> None:
    global HTTP_CLIENT
    if HTTP_CLIENT is not None:
        await HTTP_CLIENT.aclose()
        HTTP_CLIENT = None
    stop_go_process()


@app.get("/health")
async def healthcheck(request: Request) -> Response:
    try:
        ensure_go_process()
        return await proxy_request(request, "health")
    except Exception as exc:  # pragma: no cover - startup diagnostics
        return JSONResponse(status_code=503, content={"status": "error", "detail": str(exc)})


@app.get("/api/health")
async def api_healthcheck(request: Request) -> Response:
    return await healthcheck(request)


@app.get("/api/v1/billing/plans")
async def billing_plans() -> Response:
    return JSONResponse(
        {
            "plans": [
                {
                    **plan,
                    "amount_display": f"${plan['amount']:.0f}",
                    "interval": "month",
                }
                for plan in PLANS.values()
            ]
        }
    )


@app.post("/api/v1/billing/checkout")
async def billing_checkout(payload: CheckoutRequestBody, request: Request) -> Response:
    if payload.plan_id not in PLANS:
        return JSONResponse(status_code=400, content={"error": "Invalid plan selected"})

    plan = PLANS[payload.plan_id]
    origin = normalize_origin(payload.origin_url, request)
    success_url = f"{origin}/pricing?session_id={{CHECKOUT_SESSION_ID}}"
    cancel_url = f"{origin}/pricing"
    metadata = {
        "plan_id": plan["id"],
        "plan_name": plan["name"],
        "user_email": payload.user_email or "",
        "source": "gengowatcher_pricing",
    }

    checkout = get_stripe_checkout(request)
    checkout_request = CheckoutSessionRequest(
        amount=float(plan["amount"]),
        currency=plan["currency"],
        success_url=success_url,
        cancel_url=cancel_url,
        metadata=metadata,
    )
    session = await checkout.create_checkout_session(checkout_request)

    upsert_transaction(
        session_id=session.session_id,
        plan_id=plan["id"],
        plan_name=plan["name"],
        amount=float(plan["amount"]),
        currency=plan["currency"],
        status="open",
        payment_status="pending",
        user_email=payload.user_email,
        metadata=metadata,
        checkout_url=session.url,
    )

    return JSONResponse({"url": session.url, "session_id": session.session_id})


@app.get("/api/v1/billing/status/{session_id}")
async def billing_status(session_id: str, request: Request) -> Response:
    transaction = get_transaction(session_id)

    try:
        checkout = get_stripe_checkout(request)
        stripe_status = await checkout.get_checkout_status(session_id)
    except Exception as exc:
        if transaction:
            return JSONResponse(
                {
                    "session_id": session_id,
                    "status": transaction["status"],
                    "payment_status": transaction["payment_status"],
                    "amount_total": int(float(transaction["amount"]) * 100),
                    "currency": transaction["currency"],
                    "transaction": transaction,
                    "detail": "Status temporarily unavailable from Stripe; using stored transaction state.",
                }
            )
        return JSONResponse(status_code=502, content={"error": "Unable to fetch billing status", "detail": str(exc)})

    plan_id = transaction["plan_id"] if transaction else "pro"
    plan = PLANS.get(plan_id, PLANS["pro"])
    metadata = stripe_status.metadata or (transaction["metadata"] if transaction else {})

    upsert_transaction(
        session_id=session_id,
        plan_id=plan["id"],
        plan_name=plan["name"],
        amount=(stripe_status.amount_total or int(plan["amount"] * 100)) / 100,
        currency=stripe_status.currency,
        status=stripe_status.status,
        payment_status=stripe_status.payment_status,
        user_email=(transaction or {}).get("user_email"),
        metadata=metadata,
        checkout_url=None,
    )

    updated = get_transaction(session_id)
    return JSONResponse(
        {
            "session_id": session_id,
            "status": stripe_status.status,
            "payment_status": stripe_status.payment_status,
            "amount_total": stripe_status.amount_total,
            "currency": stripe_status.currency,
            "transaction": updated,
        }
    )


@app.post("/api/webhook/stripe")
async def stripe_webhook(request: Request) -> Response:
    body = await request.body()
    checkout = get_stripe_checkout(request)
    webhook_response = await checkout.handle_webhook(body, request.headers.get("Stripe-Signature"))
    transaction = get_transaction(webhook_response.session_id)

    if transaction:
        upsert_transaction(
            session_id=webhook_response.session_id,
            plan_id=transaction["plan_id"],
            plan_name=transaction["plan_name"],
            amount=transaction["amount"],
            currency=transaction["currency"],
            status=webhook_response.event_type,
            payment_status=webhook_response.payment_status,
            user_email=transaction.get("user_email"),
            metadata=webhook_response.metadata or transaction.get("metadata", {}),
            checkout_url=None,
        )

    return JSONResponse(
        {
            "received": True,
            "event_type": webhook_response.event_type,
            "session_id": webhook_response.session_id,
            "payment_status": webhook_response.payment_status,
        }
    )


@app.api_route("/{path:path}", methods=["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS", "HEAD"])
async def catch_all(path: str, request: Request) -> Response:
    try:
        ensure_go_process()
        return await proxy_request(request, path)
    except Exception as exc:
        return JSONResponse(
            status_code=502,
            content={"error": "Backend unavailable", "detail": str(exc), "log": read_go_log()},
        )