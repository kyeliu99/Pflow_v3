"""Route registration for workflow endpoints."""
from __future__ import annotations

from django.urls import include, path
from rest_framework.routers import DefaultRouter

from .views import WorkflowViewSet, health

router = DefaultRouter()
router.register("workflows", WorkflowViewSet, basename="workflow")

urlpatterns = [
    path("healthz/", health, name="workflow-health"),
    path("", include(router.urls)),
]
