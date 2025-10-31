# Generated manually for initial schema.
from __future__ import annotations

from django.db import migrations, models
import django.db.models.deletion


class Migration(migrations.Migration):
    initial = True

    dependencies = []

    operations = [
        migrations.CreateModel(
            name="Workflow",
            fields=[
                ("id", models.BigAutoField(auto_created=True, primary_key=True, serialize=False, verbose_name="ID")),
                ("name", models.CharField(max_length=255, unique=True)),
                ("description", models.TextField(blank=True)),
                ("version", models.PositiveIntegerField(default=1)),
                ("is_active", models.BooleanField(default=True)),
                ("definition", models.JSONField(blank=True, default=dict)),
                ("created_at", models.DateTimeField(auto_now_add=True)),
                ("updated_at", models.DateTimeField(auto_now=True)),
            ],
            options={"ordering": ["name", "version"]},
        ),
        migrations.CreateModel(
            name="WorkflowStep",
            fields=[
                ("id", models.BigAutoField(auto_created=True, primary_key=True, serialize=False, verbose_name="ID")),
                ("name", models.CharField(max_length=255)),
                ("step_type", models.CharField(choices=[("manual", "Manual"), ("automated", "Automated")], default="manual", max_length=32)),
                ("sequence", models.PositiveIntegerField(default=1)),
                ("metadata", models.JSONField(blank=True, default=dict)),
                (
                    "workflow",
                    models.ForeignKey(on_delete=django.db.models.deletion.CASCADE, related_name="steps", to="workflows.workflow"),
                ),
            ],
            options={"ordering": ["sequence", "id"], "unique_together": {("workflow", "sequence")}},
        ),
    ]
