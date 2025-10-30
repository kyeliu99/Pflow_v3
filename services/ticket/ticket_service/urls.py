"""URL configuration for the ticket service."""
from django.urls import include, path

urlpatterns = [
    path("api/", include("tickets.urls")),
]
