# Generated manually for initial schema.
from __future__ import annotations

from django.db import migrations, models


class Migration(migrations.Migration):
    initial = True

    dependencies = []

    operations = [
        migrations.CreateModel(
            name="User",
            fields=[
                ("id", models.BigAutoField(auto_created=True, primary_key=True, serialize=False, verbose_name="ID")),
                ("email", models.EmailField(max_length=254, unique=True)),
                ("display_name", models.CharField(max_length=255)),
                (
                    "role",
                    models.CharField(
                        choices=[("admin", "Administrator"), ("agent", "Agent"), ("requester", "Requester")],
                        default="requester",
                        max_length=32,
                    ),
                ),
                ("is_active", models.BooleanField(default=True)),
                ("created_at", models.DateTimeField(auto_now_add=True)),
                ("updated_at", models.DateTimeField(auto_now=True)),
            ],
            options={"ordering": ["display_name", "email"]},
        ),
    ]
