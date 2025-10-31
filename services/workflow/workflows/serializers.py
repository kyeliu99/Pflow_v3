"""Serializers for workflow entities."""
from __future__ import annotations

from rest_framework import serializers

from .models import Workflow, WorkflowStep


class WorkflowStepSerializer(serializers.ModelSerializer):
    class Meta:
        model = WorkflowStep
        fields = [
            "id",
            "name",
            "step_type",
            "sequence",
            "metadata",
        ]


class WorkflowSerializer(serializers.ModelSerializer):
    steps = WorkflowStepSerializer(many=True)

    class Meta:
        model = Workflow
        fields = [
            "id",
            "name",
            "description",
            "version",
            "is_active",
            "definition",
            "steps",
            "created_at",
            "updated_at",
        ]

    def create(self, validated_data):  # type: ignore[override]
        steps = validated_data.pop("steps", [])
        workflow = Workflow.objects.create(**validated_data)
        for index, step in enumerate(steps, start=1):
            WorkflowStep.objects.create(workflow=workflow, sequence=index, **step)
        return workflow

    def update(self, instance, validated_data):  # type: ignore[override]
        steps = validated_data.pop("steps", None)
        for attr, value in validated_data.items():
            setattr(instance, attr, value)
        instance.save()

        if steps is not None:
            instance.steps.all().delete()
            for index, step in enumerate(steps, start=1):
                WorkflowStep.objects.create(workflow=instance, sequence=index, **step)
        return instance
