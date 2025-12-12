# Example: API endpoint skeleton
# Claude creates this, local LLM implements the TODOs

from fastapi import APIRouter, HTTPException, Depends
from pydantic import BaseModel
from typing import Optional
from datetime import datetime

router = APIRouter(prefix="/items", tags=["items"])


class ItemCreate(BaseModel):
    name: str
    description: Optional[str] = None
    price: float
    quantity: int = 0


class ItemResponse(BaseModel):
    id: int
    name: str
    description: Optional[str]
    price: float
    quantity: int
    created_at: datetime


# In-memory storage (replace with database in production)
items_db: dict[int, dict] = {}
next_id: int = 1


@router.post("/", response_model=ItemResponse)
async def create_item(item: ItemCreate):
    """Create a new item."""
    global next_id

    # TODO: Validate that price is positive (raise HTTPException 400 if not)

    # TODO: Validate that name is not empty after stripping whitespace

    # TODO: Create item dict with id, all fields from item, and created_at=datetime.now()

    # TODO: Store in items_db using id as key

    # TODO: Increment next_id

    # TODO: Return ItemResponse with the created item data
    pass


@router.get("/{item_id}", response_model=ItemResponse)
async def get_item(item_id: int):
    """Get an item by ID."""
    # TODO: Check if item_id exists in items_db

    # TODO: If not found, raise HTTPException 404 with detail "Item not found"

    # TODO: Return ItemResponse with the item data
    pass


@router.get("/", response_model=list[ItemResponse])
async def list_items(skip: int = 0, limit: int = 10):
    """List all items with pagination."""
    # TODO: Get all values from items_db as a list

    # TODO: Apply skip and limit for pagination

    # TODO: Return list of ItemResponse objects
    pass


@router.delete("/{item_id}")
async def delete_item(item_id: int):
    """Delete an item."""
    # TODO: Check if item_id exists in items_db

    # TODO: If not found, raise HTTPException 404

    # TODO: Delete from items_db

    # TODO: Return {"status": "deleted", "id": item_id}
    pass
