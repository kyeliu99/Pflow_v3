"""API views for managing tickets."""
from __future__ import annotations

from rest_framework import viewsets
from rest_framework.decorators import action, api_view
from rest_framework.filters import OrderingFilter, SearchFilter
from rest_framework.response import Response

from .models import Ticket
from .serializers import TicketSerializer


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


@api_view(["GET"])
def health(request):  # type: ignore[override]
    """Readiness endpoint for orchestration tooling."""

    return Response({"status": "ok"})
