"""Authentication API routes for GengoWatcher SaaS.

Implements /api/v1/auth/* endpoints for registration, login, logout,
token refresh, and user management.
"""

from fastapi import APIRouter, Depends, HTTPException, Response, status
from pydantic import BaseModel, EmailStr, Field
from sqlalchemy.ext.asyncio import AsyncSession

from src.gengowatcher.auth.exceptions import AuthException, InvalidTokenException
from src.gengowatcher.auth.security import decode_token
from src.gengowatcher.auth.service import AuthService
from src.gengowatcher.database.session import get_db
from src.gengowatcher.database.models import User

router = APIRouter(prefix="/api/v1/auth", tags=["auth"])


# =============================================================================
# Request/Response Models
# =============================================================================


class RegisterRequest(BaseModel):
    """Request model for user registration."""

    email: EmailStr
    password: str = Field(..., min_length=8, max_length=100)


class LoginRequest(BaseModel):
    """Request model for user login."""

    email: EmailStr
    password: str


class RefreshRequest(BaseModel):
    """Request model for token refresh."""

    refresh_token: str


class ChangePasswordRequest(BaseModel):
    """Request model for password change."""

    old_password: str
    new_password: str = Field(..., min_length=8, max_length=100)


class AuthResponse(BaseModel):
    """Response model for successful authentication."""

    access_token: str
    refresh_token: str
    user: "UserResponse"


class UserResponse(BaseModel):
    """Response model for user data."""

    id: str
    email: str
    email_verified: bool
    is_active: bool

    @classmethod
    def from_user(cls, user: User) -> "UserResponse":
        """Create UserResponse from User model."""
        return cls(
            id=str(user.id),
            email=user.email,
            email_verified=user.email_verified,
            is_active=user.is_active,
        )


# Update forward reference
AuthResponse.model_rebuild()


class ErrorResponse(BaseModel):
    """Standard error response format."""

    error: str
    code: str
    details: dict = {}


# =============================================================================
# Dependencies
# =============================================================================


async def get_current_user(
    authorization: str = Depends(lambda: ""),
    db: AsyncSession = Depends(get_db),
) -> User:
    """Get current user from Bearer token.

    Raises:
        HTTPException: If token is invalid
    """
    if not authorization.startswith("Bearer "):
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail={"error": "Missing authorization header", "code": "MISSING_TOKEN"},
        )

    token = authorization.split(" ")[1]
    payload = decode_token(token)

    if not payload or payload.get("type") != "access":
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail={"error": "Invalid access token", "code": "INVALID_TOKEN"},
        )

    service = AuthService(db)
    user = await service.get_user_by_id(payload["sub"])

    if not user:
        raise HTTPException(
            status_code=status.HTTP_401_UNAUTHORIZED,
            detail={"error": "User not found", "code": "USER_NOT_FOUND"},
        )

    if not user.is_active:
        raise HTTPException(
            status_code=status.HTTP_403_FORBIDDEN,
            detail={"error": "User account is inactive", "code": "INACTIVE_USER"},
        )

    return user


# =============================================================================
# Routes
# =============================================================================


@router.post(
    "/register",
    response_model=AuthResponse,
    status_code=status.HTTP_201_CREATED,
)
async def register(req: RegisterRequest, db: AsyncSession = Depends(get_db)):
    """Register a new user account.

    Creates user, watcher config, and watcher state.
    Returns access and refresh tokens.
    """
    try:
        service = AuthService(db)
        user = await service.register_user(req.email, req.password)
        _, access, refresh = await service.authenticate_user(req.email, req.password)

        return AuthResponse(
            access_token=access,
            refresh_token=refresh,
            user=UserResponse.from_user(user),
        )
    except AuthException as e:
        raise HTTPException(status_code=e.status_code, detail=e.to_dict())


@router.post("/login", response_model=AuthResponse)
async def login(
    req: LoginRequest,
    db: AsyncSession = Depends(get_db),
    response: Response = None,
):
    """Authenticate user with email/password.

    Sets httpOnly cookie with refresh token and returns access token.
    """
    try:
        service = AuthService(db)
        user, access, refresh = await service.authenticate_user(req.email, req.password)

        # Set httpOnly cookie for refresh token
        if response:
            response.set_cookie(
                key="session_token",
                value=refresh,
                httponly=True,
                secure=False,  # Set True in production with HTTPS
                samesite="lax",
                max_age=7 * 24 * 60 * 60,  # 7 days
            )

        return AuthResponse(
            access_token=access,
            refresh_token=refresh,
            user=UserResponse.from_user(user),
        )
    except AuthException as e:
        raise HTTPException(status_code=e.status_code, detail=e.to_dict())


@router.post("/logout", status_code=status.HTTP_204_NO_CONTENT)
async def logout(
    response: Response,
    refresh_token: str = None,
    db: AsyncSession = Depends(get_db),
):
    """Logout user and revoke refresh token.

    Accepts token from request body or httpOnly cookie.
    """
    # Try cookie first, then body
    token = refresh_token  # TODO: Also check cookie when we have cookie access

    if token:
        service = AuthService(db)
        await service.revoke_refresh_token(token)

    # Clear cookie
    response.delete_cookie("session_token")
    return Response(status_code=status.HTTP_204_NO_CONTENT)


@router.post("/refresh", response_model=AuthResponse)
async def refresh(
    req: RefreshRequest,
    db: AsyncSession = Depends(get_db),
):
    """Refresh access token using refresh token.

    Implements token rotation - returns new refresh token
    and revokes old one.
    """
    try:
        service = AuthService(db)
        access, refresh, user = await service.refresh_tokens(req.refresh_token)

        return AuthResponse(
            access_token=access,
            refresh_token=refresh,
            user=UserResponse.from_user(user),
        )
    except AuthException as e:
        raise HTTPException(status_code=e.status_code, detail=e.to_dict())


@router.get("/me", response_model=UserResponse)
async def get_me(current_user: User = Depends(get_current_user)):
    """Get current authenticated user info."""
    return UserResponse.from_user(current_user)


@router.post("/change-password", status_code=status.HTTP_204_NO_CONTENT)
async def change_password(
    req: ChangePasswordRequest,
    current_user: User = Depends(get_current_user),
    db: AsyncSession = Depends(get_db),
):
    """Change current user's password.

    Requires old password for verification.
    Revokes all existing refresh tokens after change.
    """
    try:
        service = AuthService(db)
        await service.change_password(current_user.id, req.old_password, req.new_password)
        return Response(status_code=status.HTTP_204_NO_CONTENT)
    except AuthException as e:
        raise HTTPException(status_code=e.status_code, detail=e.to_dict())


# =============================================================================
# Exception Handlers
# =============================================================================


@router.exception_handler(AuthException)
async def auth_exception_handler(request, exc: AuthException):
    """Handle AuthException and return standard error format."""
    from fastapi.responses import JSONResponse

    return JSONResponse(status_code=exc.status_code, content=exc.to_dict())
