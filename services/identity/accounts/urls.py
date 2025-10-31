"""Route registration for identity endpoints."""
from __future__ import annotations

from django.urls import include, path
from rest_framework.routers import DefaultRouter

from .views import UserViewSet, health

router = DefaultRouter()
router.register("users", UserViewSet, basename="user")

urlpatterns = [
    path("healthz/", health, name="identity-health"),
    path("", include(router.urls)),
]
