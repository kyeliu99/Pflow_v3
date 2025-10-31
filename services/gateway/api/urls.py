"""Gateway API routes."""
from __future__ import annotations

from django.urls import path

from . import views

urlpatterns = [
    path("healthz/", views.health, name="gateway-health"),
    path("overview/", views.overview, name="gateway-overview"),
    path("forms/", views.FormCollectionView.as_view(), name="gateway-forms"),
    path("forms/<int:resource_id>/", views.FormDetailView.as_view(), name="gateway-form-detail"),
    path("users/", views.UserCollectionView.as_view(), name="gateway-users"),
    path("users/<int:resource_id>/", views.UserDetailView.as_view(), name="gateway-user-detail"),
    path("tickets/", views.TicketCollectionView.as_view(), name="gateway-tickets"),
    path("tickets/<int:resource_id>/", views.TicketDetailView.as_view(), name="gateway-ticket-detail"),
    path(
        "tickets/submissions/",
        views.TicketSubmissionCollectionView.as_view(),
        name="gateway-ticket-submissions",
    ),
    path(
        "tickets/submissions/<uuid:submission_id>/",
        views.TicketSubmissionDetailView.as_view(),
        name="gateway-ticket-submission-detail",
    ),
    path(
        "tickets/queue-metrics/",
        views.TicketQueueMetricsView.as_view(),
        name="gateway-ticket-queue-metrics",
    ),
    path("workflows/", views.WorkflowCollectionView.as_view(), name="gateway-workflows"),
    path("workflows/<int:resource_id>/", views.WorkflowDetailView.as_view(), name="gateway-workflow-detail"),
]
