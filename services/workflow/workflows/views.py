"""API views for workflows."""
from __future__ import annotations

from rest_framework import viewsets
from rest_framework.decorators import action, api_view
from rest_framework.filters import OrderingFilter, SearchFilter
from rest_framework.response import Response

from .models import Workflow
from .serializers import WorkflowSerializer


class WorkflowViewSet(viewsets.ModelViewSet):
    queryset = Workflow.objects.prefetch_related("steps").all()
    serializer_class = WorkflowSerializer
    filter_backends = [SearchFilter, OrderingFilter]
    search_fields = ["name", "description"]
    ordering_fields = ["name", "updated_at"]
    ordering = ["name"]

    @action(detail=True, methods=["post"], url_path="publish")
    def publish(self, request, *args, **kwargs):  # type: ignore[override]
        """Activate a workflow definition."""

        workflow = self.get_object()
        workflow.is_active = True
        workflow.save(update_fields=["is_active", "updated_at"])
        serializer = self.get_serializer(workflow)
        return Response(serializer.data)


@api_view(["GET"])
def health(request):  # type: ignore[override]
    """Readiness endpoint for orchestration tooling."""

    return Response({"status": "ok"})
