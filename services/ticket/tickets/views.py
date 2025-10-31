"""API views for managing tickets."""
from __future__ import annotations

import uuid
from typing import Dict

from django.db.models import Count
from django.utils import timezone
from rest_framework import mixins, status, viewsets
from rest_framework.decorators import action, api_view
from rest_framework.filters import OrderingFilter, SearchFilter
from rest_framework.request import Request
from rest_framework.response import Response

from .models import Ticket, TicketSubmission
from .serializers import (
    TicketSerializer,
    TicketSubmissionRequestSerializer,
    TicketSubmissionSerializer,
)
from .tasks import process_ticket_submission


class TicketViewSet(viewsets.ModelViewSet):
    queryset = Ticket.objects.all()
    serializer_class = TicketSerializer
    filter_backends = [SearchFilter, OrderingFilter]
    search_fields = ["title", "description", "status", "priority"]
    ordering_fields = ["created_at", "updated_at", "priority"]
    ordering = ["-created_at"]

    @action(detail=True, methods=["post"], url_path="resolve")
    def resolve(self, request, *args, **kwargs):  # type: ignore[override]
        """Mark a ticket as resolved."""

        ticket = self.get_object()
        ticket.status = Ticket.RESOLVED
        ticket.save(update_fields=["status", "updated_at"])
        serializer = self.get_serializer(ticket)
        return Response(serializer.data)


class TicketSubmissionViewSet(
    mixins.CreateModelMixin,
    mixins.RetrieveModelMixin,
    viewsets.GenericViewSet,
):
    queryset = TicketSubmission.objects.select_related("ticket").all()
    serializer_class = TicketSubmissionSerializer
    lookup_field = "id"
    lookup_value_regex = "[0-9a-f\-]+"

    def create(self, request: Request, *args, **kwargs):  # type: ignore[override]
        payload_serializer = TicketSubmissionRequestSerializer(data=request.data)
        payload_serializer.is_valid(raise_exception=True)
        data = payload_serializer.validated_data

        client_reference = data.get("client_reference")
        submission: TicketSubmission | None = None
        if client_reference is not None:
            submission = TicketSubmission.objects.filter(client_reference=client_reference).first()
            if submission:
                if submission.status == TicketSubmission.FAILED:
                    submission.status = TicketSubmission.PENDING
                    submission.error_message = ""
                    submission.ticket = None
                    submission.completed_at = None
                    submission.request_payload = {
                        key: value for key, value in data.items() if key != "client_reference"
                    }
                    submission.save(
                        update_fields=[
                            "status",
                            "error_message",
                            "ticket",
                            "completed_at",
                            "request_payload",
                            "updated_at",
                        ]
                    )
                serializer = self.get_serializer(submission)
                if submission.status in {TicketSubmission.PENDING, TicketSubmission.PROCESSING}:
                    process_ticket_submission.delay(str(submission.id))
                status_code = (
                    status.HTTP_200_OK
                    if submission.status == TicketSubmission.COMPLETED
                    else status.HTTP_202_ACCEPTED
                )
                return Response(serializer.data, status=status_code)

        if submission is None:
            submission = TicketSubmission.objects.create(
                client_reference=client_reference or uuid.uuid4(),
                request_payload={
                    key: value for key, value in data.items() if key != "client_reference"
                },
            )

        process_ticket_submission.delay(str(submission.id))
        serializer = self.get_serializer(submission)
        return Response(serializer.data, status=status.HTTP_202_ACCEPTED)


@api_view(["GET"])
def health(request: Request):  # type: ignore[override]
    """Readiness endpoint for orchestration tooling."""

    return Response({"status": "ok"})


@api_view(["GET"])
def queue_metrics(_: Request) -> Response:
    """Provide observability data for the submission queue."""

    totals: Dict[str, int] = {
        TicketSubmission.PENDING: 0,
        TicketSubmission.PROCESSING: 0,
        TicketSubmission.COMPLETED: 0,
        TicketSubmission.FAILED: 0,
    }

    for entry in (
        TicketSubmission.objects.values("status").order_by().annotate(total=Count("id"))
    ):
        status_value = entry.get("status")
        if status_value in totals:
            totals[status_value] = int(entry.get("total", 0))

    oldest_pending = (
        TicketSubmission.objects.filter(
            status__in=[TicketSubmission.PENDING, TicketSubmission.PROCESSING]
        )
        .order_by("created_at")
        .first()
    )
    if oldest_pending is not None:
        wait_seconds = max(
            int((timezone.now() - oldest_pending.created_at).total_seconds()),
            0,
        )
    else:
        wait_seconds = 0

    return Response(
        {
            "pending": totals[TicketSubmission.PENDING],
            "processing": totals[TicketSubmission.PROCESSING],
            "completed": totals[TicketSubmission.COMPLETED],
            "failed": totals[TicketSubmission.FAILED],
            "oldestPendingSeconds": wait_seconds,
        }
    )
