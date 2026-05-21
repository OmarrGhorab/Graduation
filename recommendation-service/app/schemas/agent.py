from typing import Dict, List, Literal, Optional
from pydantic import BaseModel, Field


EmbeddingEntityType = Literal["course", "user", "cluster"]


class EmbeddingRequest(BaseModel):
    text: str = Field(..., min_length=1, max_length=12000)
    entity_type: EmbeddingEntityType
    entity_id: Optional[str] = None
    normalize: bool = True


class BatchEmbeddingRequest(BaseModel):
    texts: List[str] = Field(..., min_length=1, max_length=256)
    normalize: bool = True


class EmbeddingVector(BaseModel):
    vector: List[float]
    dimensions: int
    model: str


class EmbeddingDocument(BaseModel):
    entity_type: EmbeddingEntityType
    entity_id: str
    vector: List[float]
    metadata: Dict = Field(default_factory=dict)
    tags: List[str] = Field(default_factory=list)
