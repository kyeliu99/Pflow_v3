# Generated manually for initial schema.
from __future__ import annotations

from django.db import migrations, models
import django.db.models.deletion


class Migration(migrations.Migration):
    initial = True

    dependencies = []

    operations = [
        migrations.CreateModel(
            name="Form",
            fields=[
                ("id", models.BigAutoField(auto_created=True, primary_key=True, serialize=False, verbose_name="ID")),
                ("name", models.CharField(max_length=255)),
                ("description", models.TextField(blank=True)),
                ("version", models.PositiveIntegerField(default=1)),
                ("is_active", models.BooleanField(default=True)),
                ("created_at", models.DateTimeField(auto_now_add=True)),
                ("updated_at", models.DateTimeField(auto_now=True)),
            ],
            options={"ordering": ["name", "id"]},
        ),
        migrations.CreateModel(
            name="FormField",
            fields=[
                ("id", models.BigAutoField(auto_created=True, primary_key=True, serialize=False, verbose_name="ID")),
                ("name", models.CharField(max_length=255)),
                ("label", models.CharField(max_length=255)),
                ("field_type", models.CharField(choices=[("text", "Text"), ("number", "Number"), ("date", "Date"), ("select", "Select")], max_length=32)),
                ("required", models.BooleanField(default=False)),
                ("order", models.PositiveIntegerField(default=0)),
                ("metadata", models.JSONField(blank=True, default=dict)),
                (
                    "form",
                    models.ForeignKey(on_delete=django.db.models.deletion.CASCADE, related_name="fields", to="forms.form"),
                ),
            ],
            options={"ordering": ["order", "id"], "unique_together": {("form", "name")}},
        ),
    ]
