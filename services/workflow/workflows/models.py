"""Database models for the workflow service."""
from __future__ import annotations

from django.db import models


class Workflow(models.Model):
    """A workflow definition that orchestrates ticket progress."""

    name = models.CharField(max_length=255, unique=True)
    description = models.TextField(blank=True)
    version = models.PositiveIntegerField(default=1)
    is_active = models.BooleanField(default=True)
    definition = models.JSONField(default=dict, blank=True)
    created_at = models.DateTimeField(auto_now_add=True)
    updated_at = models.DateTimeField(auto_now=True)

    class Meta:
        ordering = ["name", "version"]

    def __str__(self) -> str:
        return f"{self.name} v{self.version}"


class WorkflowStep(models.Model):
    """An ordered step inside a workflow."""

    MANUAL = "manual"
    AUTOMATED = "automated"

    STEP_TYPES = [
        (MANUAL, "Manual"),
        (AUTOMATED, "Automated"),
    ]

    workflow = models.ForeignKey(Workflow, related_name="steps", on_delete=models.CASCADE)
    name = models.CharField(max_length=255)
    step_type = models.CharField(max_length=32, choices=STEP_TYPES, default=MANUAL)
    sequence = models.PositiveIntegerField(default=1)
    metadata = models.JSONField(default=dict, blank=True)

    class Meta:
        ordering = ["sequence", "id"]
        unique_together = ("workflow", "sequence")

    def __str__(self) -> str:
        return f"{self.sequence}. {self.name}"
