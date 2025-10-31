"""URL configuration for the workflow service."""
from django.urls import include, path

urlpatterns = [
    path("api/", include("workflows.urls")),
]
