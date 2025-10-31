"""Database models for the ticket service."""
from __future__ import annotations

import uuid

from django.db import models
from django.utils import timezone


class Ticket(models.Model):
    """A workflow ticket raised from a form submission."""

    DRAFT = "draft"
    OPEN = "open"
    IN_PROGRESS = "in_progress"
    RESOLVED = "resolved"
    CLOSED = "closed"

    STATUS_CHOICES = [
        (DRAFT, "Draft"),
        (OPEN, "Open"),
        (IN_PROGRESS, "In Progress"),
        (RESOLVED, "Resolved"),
        (CLOSED, "Closed"),
    ]

    LOW = "low"
    MEDIUM = "medium"
    HIGH = "high"

    PRIORITY_CHOICES = [
        (LOW, "Low"),
        (MEDIUM, "Medium"),
        (HIGH, "High"),
    ]

    title = models.CharField(max_length=255)
    description = models.TextField(blank=True)
    status = models.CharField(max_length=32, choices=STATUS_CHOICES, default=OPEN)
    priority = models.CharField(max_length=16, choices=PRIORITY_CHOICES, default=MEDIUM)
    form_id = models.IntegerField()
    requester_id = models.IntegerField(null=True, blank=True)
    assignee_id = models.IntegerField(null=True, blank=True)
    workflow_id = models.IntegerField(null=True, blank=True)
    payload = models.JSONField(default=dict, blank=True)
    due_date = models.DateField(null=True, blank=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        ordering = ["-created_at", "id"]

    def __str__(self) -> str:
        return f"{self.title} ({self.status})"


class TicketSubmission(models.Model):
    """Queue-backed submission used to create tickets under load."""

    PENDING = "pending"
    PROCESSING = "processing"
    COMPLETED = "completed"
    FAILED = "failed"

    STATUS_CHOICES = [
        (PENDING, "Pending"),
        (PROCESSING, "Processing"),
        (COMPLETED, "Completed"),
        (FAILED, "Failed"),
    ]

    id = models.UUIDField(primary_key=True, default=uuid.uuid4, editable=False)
    client_reference = models.UUIDField(default=uuid.uuid4, unique=True, editable=False)
    status = models.CharField(max_length=32, choices=STATUS_CHOICES, default=PENDING)
    ticket = models.ForeignKey(
        "Ticket",
        on_delete=models.SET_NULL,
        related_name="submissions",
        null=True,
        blank=True,
    )
    request_payload = models.JSONField(default=dict)
    error_message = models.TextField(blank=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)
    completed_at = models.DateTimeField(null=True, blank=True)

    class Meta:
        ordering = ["-created_at"]
        indexes = [
            models.Index(fields=["status"]),
            models.Index(fields=["client_reference"]),
        ]

    def mark_processing(self) -> None:
        self.status = self.PROCESSING
        self.save(update_fields=["status", "updated_at"])

    def mark_completed(self, ticket: Ticket) -> None:
        self.ticket = ticket
        self.status = self.COMPLETED
        self.completed_at = timezone.now()
        self.error_message = ""
        self.save(update_fields=["ticket", "status", "completed_at", "error_message", "updated_at"])

    def mark_failed(self, message: str) -> None:
        self.status = self.FAILED
        self.error_message = message
        self.completed_at = timezone.now()
        self.save(update_fields=["status", "error_message", "completed_at", "updated_at"])
