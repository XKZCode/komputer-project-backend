from fastapi import FastAPI, UploadFile, File, HTTPException
from fastapi.middleware.cors import CORSMiddleware
from faster_whisper import WhisperModel
from scipy.io.wavfile import write
import sounddevice as sd
import requests
import shutil
import os

app = FastAPI(title="HT Audio STT Processor")

#Mengaktifkan CORS agar dapat dipanggil dari tester.html di browser
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

print("Loading Whisper STT model...")
model = WhisperModel("medium", device="cpu", compute_type="int8")
print("Whisper model loaded successfully!")

@app.post("/process-audio")
async def process_audio(file: UploadFile = File(...)):
    #Simpan audio sementara ke disk
    temp_filename = f"temp_{file.filename}"
    try:
        with open(temp_filename, "wb") as buffer:
            shutil.copyfileobj(file.file, buffer)
        
        #Proses Speech-to-Text
        segments, _ = model.transcribe(
            temp_filename,
            language="id",
            beam_size=5,
            initial_prompt="kebakaran, pencuri, pencurian, tolong, darurat, bahaya, bantuan, polisi, teror, ledakan"
        )
        
        transcript = " ".join([segment.text for segment in segments]).strip()
        
        #Hapus file sementara
        if os.path.exists(temp_filename):
            os.remove(temp_filename)
            
        return {"status": "success", "transcript": transcript}

    except Exception as e:
        if os.path.exists(temp_filename):
            os.remove(temp_filename)
        raise HTTPException(status_code=500, detail=str(e))

@app.get("/health")
def health_check():
    return {"status": "ok"}

@app.get("/record-and-send")
def record_from_web():
    duration = 5
    sample_rate = 44100
    filename = "web_input.wav"
    
    #Rekam audio dari mic laptop selama 5 detik
    audio_data = sd.rec(int(duration * sample_rate), samplerate=sample_rate, channels=1, dtype='int16')
    sd.wait()
    write(filename, sample_rate, audio_data)
    
    #Kirim otomatis ke Go Backend API
    url = "http://localhost:8080/api/audio/upload"
    payload = {"latitude": "-6.917464", "longitude": "107.619123"}
    
    try:
        with open(filename, "rb") as f:
            files = {"audio": (filename, f, "audio/wav")}
            response = requests.post(url, data=payload, files=files)
            
        return response.json()
    except Exception as e:
        raise HTTPException(status_code=500, detail=f"Gagal mengirim ke Go Backend: {str(e)}")