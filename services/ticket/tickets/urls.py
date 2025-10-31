"""Route registration for ticket endpoints."""
from __future__ import annotations

from django.urls import include, path
from rest_framework.routers import DefaultRouter

from .views import TicketSubmissionViewSet, TicketViewSet, health, queue_metrics

router = DefaultRouter()
router.register(
    "tickets/submissions",
    TicketSubmissionViewSet,
    basename="ticket-submission",
)
router.register("tickets", TicketViewSet, basename="ticket")

urlpatterns = [
    path("healthz/", health, name="ticket-health"),
    path("tickets/queue-metrics/", queue_metrics, name="ticket-queue-metrics"),
    path("", include(router.urls)),
]
