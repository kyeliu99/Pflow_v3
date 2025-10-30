"""Route registration for ticket endpoints."""
from __future__ import annotations

from django.urls import include, path
from rest_framework.routers import DefaultRouter

from .views import TicketViewSet, health

router = DefaultRouter()
router.register("tickets", TicketViewSet, basename="ticket")

urlpatterns = [
    path("healthz/", health, name="ticket-health"),
    path("", include(router.urls)),
]
