from fastapi import FastAPI, UploadFile, File, HTTPException
from fastapi.responses import JSONResponse
from pydantic import BaseModel
from typing import Optional, List
import io
import os

app = FastAPI(title="TrustVault Document Service", version="1.0.0")

class ExtractionResult(BaseModel):
    text: str
    pages: int
    format: str
    metadata: dict

class ClassificationResult(BaseModel):
    entities: List[dict]
    confidence: float

@app.get("/health")
async def health():
    return {"status": "ok", "service": "docservice"}

@app.post("/extract", response_model=ExtractionResult)
async def extract_document(file: UploadFile = File(...)):
    """Extract text from uploaded document"""
    content = await file.read()
    filename = file.filename or "unknown"
    ext = os.path.splitext(filename)[1].lower()
    
    try:
        if ext == ".pdf":
            text, pages = extract_pdf(content)
        elif ext in [".xlsx", ".xls"]:
            text, pages = extract_excel(content)
        elif ext == ".csv":
            text, pages = extract_csv(content)
        elif ext in [".png", ".jpg", ".jpeg", ".tiff"]:
            text, pages = extract_image(content)
        elif ext == ".docx":
            text, pages = extract_docx(content)
        else:
            text = content.decode("utf-8", errors="ignore")
            pages = 1
        
        return ExtractionResult(
            text=text,
            pages=pages,
            format=ext.lstrip("."),
            metadata={"filename": filename, "size": len(content)}
        )
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

@app.post("/ocr")
async def ocr_document(file: UploadFile = File(...)):
    """Run OCR on image or scanned PDF"""
    content = await file.read()
    
    try:
        text = run_ocr(content)
        return {"text": text, "confidence": 0.95}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))

def extract_pdf(content: bytes) -> tuple[str, int]:
    """Extract text from PDF using pdfplumber"""
    try:
        import pdfplumber
        text_parts = []
        pages = 0
        with pdfplumber.open(io.BytesIO(content)) as pdf:
            pages = len(pdf.pages)
            for page in pdf.pages:
                text = page.extract_text()
                if text:
                    text_parts.append(text)
        return "\n\n".join(text_parts), pages
    except ImportError:
        return "PDF extraction requires pdfplumber", 0

def extract_excel(content: bytes) -> tuple[str, int]:
    """Extract text from Excel files"""
    try:
        import openpyxl
        wb = openpyxl.load_workbook(io.BytesIO(content), data_only=True)
        text_parts = []
        for sheet in wb.worksheets:
            for row in sheet.iter_rows(values_only=True):
                row_text = " | ".join(str(cell) if cell else "" for cell in row)
                if row_text.strip():
                    text_parts.append(row_text)
        return "\n".join(text_parts), len(wb.worksheets)
    except ImportError:
        return "Excel extraction requires openpyxl", 0

def extract_csv(content: bytes) -> tuple[str, int]:
    """Extract text from CSV"""
    import csv
    text = content.decode("utf-8", errors="ignore")
    reader = csv.reader(io.StringIO(text))
    rows = [" | ".join(row) for row in reader]
    return "\n".join(rows), 1

def extract_image(content: bytes) -> tuple[str, int]:
    """Extract text from image using OCR"""
    text = run_ocr(content)
    return text, 1

def extract_docx(content: bytes) -> tuple[str, int]:
    """Extract text from DOCX"""
    try:
        from docx import Document
        doc = Document(io.BytesIO(content))
        text_parts = [para.text for para in doc.paragraphs if para.text]
        return "\n\n".join(text_parts), 1
    except ImportError:
        return "DOCX extraction requires python-docx", 0

def run_ocr(content: bytes) -> str:
    """Run OCR using PaddleOCR or Tesseract"""
    try:
        from paddleocr import PaddleOCR
        ocr = PaddleOCR(use_angle_cls=True, lang='en', show_log=False)
        
        # Save to temp file for PaddleOCR
        import tempfile
        with tempfile.NamedTemporaryFile(suffix=".png", delete=False) as f:
            f.write(content)
            temp_path = f.name
        
        result = ocr.ocr(temp_path, cls=True)
        os.unlink(temp_path)
        
        if result and result[0]:
            lines = [line[1][0] for line in result[0]]
            return "\n".join(lines)
        return ""
    except ImportError:
        # Fallback to Tesseract
        try:
            import pytesseract
            from PIL import Image
            img = Image.open(io.BytesIO(content))
            return pytesseract.image_to_string(img)
        except ImportError:
            return "OCR requires paddleocr or pytesseract"

if __name__ == "__main__":
    import uvicorn
    uvicorn.run(app, host="0.0.0.0", port=8088)
