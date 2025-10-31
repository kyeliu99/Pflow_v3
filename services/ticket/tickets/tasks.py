"""Background tasks for the ticket service."""
from __future__ import annotations

import logging
from typing import Any, Dict

from celery import shared_task
from django.db import transaction

from .models import Ticket, TicketSubmission

logger = logging.getLogger(__name__)


def _build_ticket_kwargs(payload: Dict[str, Any]) -> Dict[str, Any]:
    """Translate a submission payload into Ticket model kwargs."""

    return {
        "title": payload["title"],
        "description": payload.get("description", ""),
        "form_id": payload["form_id"],
        "assignee_id": payload.get("assignee_id"),
        "requester_id": payload.get("requester_id"),
        "priority": payload.get("priority", Ticket.MEDIUM),
        "status": payload.get("status", Ticket.OPEN),
        "payload": payload.get("payload", {}),
    }


@shared_task(bind=True, max_retries=3, default_retry_delay=5)
def process_ticket_submission(self, submission_id: str) -> None:
    """Persist a ticket asynchronously to absorb bursty traffic."""

    submission: TicketSubmission | None = None
    try:
        with transaction.atomic():
            submission = (
                TicketSubmission.objects.select_for_update().get(id=submission_id)
            )
            if submission.status == TicketSubmission.COMPLETED:
                logger.info("Submission %s already completed", submission_id)
                return
            if submission.status == TicketSubmission.PROCESSING:
                logger.info("Submission %s already processing", submission_id)
                return
            submission.mark_processing()

        payload = dict(submission.request_payload)
        ticket_kwargs = _build_ticket_kwargs(payload)
        with transaction.atomic():
            ticket = Ticket.objects.create(**ticket_kwargs)
        submission.mark_completed(ticket)
        logger.info("Ticket %s created from submission %s", ticket.id, submission_id)
    except TicketSubmission.DoesNotExist:
        logger.warning("Submission %s does not exist", submission_id)
    except Exception as exc:  # pragma: no cover - retries exercised in production
        logger.exception("Processing submission %s failed", submission_id)
        if submission is not None:
            if self.request.retries >= self.max_retries:
                submission.mark_failed(str(exc))
                return
            submission.status = TicketSubmission.PENDING
            submission.save(update_fields=["status", "updated_at"])
        raise self.retry(exc=exc, countdown=min(60, 2 ** self.request.retries))
