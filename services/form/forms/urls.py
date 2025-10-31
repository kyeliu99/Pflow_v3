"""Route registration for the form service."""
from __future__ import annotations

from django.urls import include, path
from rest_framework.routers import DefaultRouter

from .views import FormViewSet, health

router = DefaultRouter()
router.register("forms", FormViewSet, basename="form")

urlpatterns = [
    path("healthz/", health, name="form-health"),
    path("", include(router.urls)),
]
