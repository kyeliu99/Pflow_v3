"""Serializers for the form service."""
from __future__ import annotations

from rest_framework import serializers

from .models import Form, FormField


class FormFieldSerializer(serializers.ModelSerializer):
    class Meta:
        model = FormField
        fields = [
            "id",
            "name",
            "label",
            "field_type",
            "required",
            "order",
            "metadata",
        ]


class FormSerializer(serializers.ModelSerializer):
    fields = FormFieldSerializer(many=True)

    class Meta:
        model = Form
        fields = [
            "id",
            "name",
            "description",
            "version",
            "is_active",
            "created_at",
            "updated_at",
            "fields",
        ]

    def create(self, validated_data):  # type: ignore[override]
        fields = validated_data.pop("fields", [])
        form = Form.objects.create(**validated_data)
        for index, field in enumerate(fields):
            FormField.objects.create(form=form, order=index, **field)
        return form

    def update(self, instance, validated_data):  # type: ignore[override]
        fields = validated_data.pop("fields", None)
        for attr, value in validated_data.items():
            setattr(instance, attr, value)
        instance.save()

        if fields is not None:
            instance.fields.all().delete()
            for index, field in enumerate(fields):
                FormField.objects.create(form=instance, order=index, **field)
        return instance
