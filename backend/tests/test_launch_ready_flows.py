"""Launch-ready backend regression tests for auth, session, and billing flows."""

import os
import time
import uuid

import pytest
import requests


BASE_URL = os.environ.get("REACT_APP_BACKEND_URL")


@pytest.fixture(scope="session")
def api_base_url() -> str:
    if not BASE_URL:
        pytest.fail("REACT_APP_BACKEND_URL is required for public endpoint testing")
    return BASE_URL.rstrip("/")


@pytest.fixture()
def api_client() -> requests.Session:
    session = requests.Session()
    session.headers.update({"Content-Type": "application/json"})
    return session


# Auth module: register/login/session status endpoints
def test_health_endpoint(api_client: requests.Session, api_base_url: str) -> None:
    response = api_client.get(f"{api_base_url}/api/health")
    assert response.status_code == 200
    data = response.json()
    assert isinstance(data, dict)
    assert data.get("status") == "ok"


def test_register_login_and_me_flow(api_client: requests.Session, api_base_url: str) -> None:
    email = f"launchtester+{uuid.uuid4().hex[:8]}@example.com"
    password = "LaunchReady123!"

    register_response = api_client.post(
        f"{api_base_url}/api/v1/auth/register",
        json={"email": email, "password": password},
    )
    assert register_response.status_code in (200, 201)

    register_data = register_response.json()
    assert isinstance(register_data.get("access_token"), str)
    assert register_data["user"]["email"] == email

    api_client.headers.update({"Authorization": f"Bearer {register_data['access_token']}"})
    me_after_register = api_client.get(f"{api_base_url}/api/v1/me")
    assert me_after_register.status_code == 200
    me_data = me_after_register.json()
    assert me_data.get("email") == email

    login_response = requests.post(
        f"{api_base_url}/api/v1/auth/login",
        json={"email": email, "password": password},
        headers={"Content-Type": "application/json"},
    )
    assert login_response.status_code == 200
    login_data = login_response.json()
    assert isinstance(login_data.get("access_token"), str)
    assert login_data["user"]["email"] == email


def test_login_with_provided_credentials(api_client: requests.Session, api_base_url: str) -> None:
    response = api_client.post(
        f"{api_base_url}/api/v1/auth/login",
        json={"email": "launchtester@example.com", "password": "LaunchReady123!"},
    )
    assert response.status_code == 200
    data = response.json()
    assert data["user"]["email"] == "launchtester@example.com"
    assert isinstance(data.get("access_token"), str)


# Billing module: plans, checkout session creation, and checkout session status endpoints
def test_billing_plans(api_client: requests.Session, api_base_url: str) -> None:
    response = api_client.get(f"{api_base_url}/api/v1/billing/plans")
    assert response.status_code == 200
    data = response.json()
    assert isinstance(data.get("plans"), list)
    assert len(data["plans"]) >= 2

    plan_ids = {plan.get("id") for plan in data["plans"]}
    assert "pro" in plan_ids
    assert "team" in plan_ids


def test_checkout_create_and_status(api_client: requests.Session, api_base_url: str) -> None:
    checkout_response = api_client.post(
        f"{api_base_url}/api/v1/billing/checkout",
        json={
            "plan_id": "pro",
            "origin_url": api_base_url,
            "user_email": "launchtester@example.com",
        },
    )
    assert checkout_response.status_code == 200
    checkout_data = checkout_response.json()
    assert isinstance(checkout_data.get("url"), str)
    assert checkout_data["url"].startswith("http")
    assert isinstance(checkout_data.get("session_id"), str)

    session_id = checkout_data["session_id"]
    time.sleep(1)
    status_response = api_client.get(f"{api_base_url}/api/v1/billing/status/{session_id}")
    assert status_response.status_code == 200
    status_data = status_response.json()
    assert status_data.get("session_id") == session_id
    assert status_data.get("status") in {"open", "complete", "expired"}
    assert status_data.get("payment_status") in {"paid", "unpaid", "no_payment_required", "pending"}
