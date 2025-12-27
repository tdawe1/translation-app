"""Main FastAPI application for GengoWatcher SaaS."""

from fastapi import FastAPI
from fastapi.middleware.cors import CORSMiddleware

from src.gengowatcher.auth.routes import router as auth_router

# Create FastAPI app
app = FastAPI(
    title="GengoWatcher SaaS",
    description="Multi-tenant job monitoring SaaS with per-user watcher instances",
    version="0.1.0",
    docs_url="/docs",
    redoc_url="/redoc",
)

# CORS middleware
app.add_middleware(
    CORSMiddleware,
    allow_origins=["http://localhost:5173", "http://127.0.0.1:5173"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

# Include routers
app.include_router(auth_router)


@app.get("/health")
async def health_check():
    """Health check endpoint."""
    return {"status": "healthy", "service": "gengowatcher-saas"}


@app.get("/")
async def root():
    """Root endpoint."""
    return {
        "service": "GengoWatcher SaaS",
        "version": "0.1.0",
        "docs": "/docs",
    }


if __name__ == "__main__":
    import uvicorn

    uvicorn.run(app, host="127.0.0.1", port=8000, reload=True)
