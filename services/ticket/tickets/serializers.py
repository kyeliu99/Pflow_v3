"""Serializers for ticket entities."""
from __future__ import annotations

from typing import Any, Dict

from rest_framework import serializers

from .models import Ticket, TicketSubmission


class TicketSerializer(serializers.ModelSerializer):
    class Meta:
        model = Ticket
        fields = [
            "id",
            "title",
            "description",
            "status",
            "priority",
            "form_id",
            "requester_id",
            "assignee_id",
            "workflow_id",
            "payload",
            "due_date",
            "created_at",
            "updated_at",
        ]


class TicketSubmissionSerializer(serializers.ModelSerializer):
    ticket = TicketSerializer(read_only=True)

    class Meta:
        model = TicketSubmission
        fields = [
            "id",
            "client_reference",
            "status",
            "ticket",
            "error_message",
            "created_at",
            "updated_at",
            "completed_at",
        ]


class TicketSubmissionRequestSerializer(serializers.Serializer):
    title = serializers.CharField(max_length=255)
    description = serializers.CharField(required=False, allow_blank=True, default="")
    form_id = serializers.IntegerField()
    assignee_id = serializers.IntegerField(required=False, allow_null=True)
    requester_id = serializers.IntegerField(required=False, allow_null=True)
    status = serializers.ChoiceField(choices=Ticket.STATUS_CHOICES, required=False, default=Ticket.OPEN)
    priority = serializers.ChoiceField(choices=Ticket.PRIORITY_CHOICES, default=Ticket.MEDIUM)
    payload = serializers.JSONField(required=False, default=dict)
    client_reference = serializers.UUIDField(required=False)

    def to_internal_value(self, data: Dict[str, Any]) -> Dict[str, Any]:  # type: ignore[override]
        """Ensure payload defaults to a JSON object and strip empty metadata."""

        internal = super().to_internal_value(data)
        payload = internal.get("payload") or {}
        if not isinstance(payload, dict):
            raise serializers.ValidationError({"payload": "Must be a JSON object."})
        internal["payload"] = payload
        return internal
