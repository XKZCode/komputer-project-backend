import sounddevice as sd
from scipy.io.wavfile import write
import requests
import time

# Konfigurasi rekaman
DURATION = 5
SAMPLE_RATE = 44100
FILENAME = "real_input.wav"

print("=" * 50)
print(f"Mulai merekam suara dari mic selama {DURATION} detik...")
print("Silakan bicara, contoh: 'Tolong ada pencurian dan kebakaran di jalan!'")
print("=" * 50)

#Rekam Suara dari Mic Laptop
audio_data = sd.rec(int(DURATION * SAMPLE_RATE), samplerate=SAMPLE_RATE, channels=1, dtype='int16')
sd.wait()
print("Selesai merekam! Menyimpan file...")

#Simpan ke File WAV
write(FILENAME, SAMPLE_RATE, audio_data)

#Kirim Otomatis ke Go Backend API
print("Mengirim audio ke Go Backend API...")
url = "http://localhost:8080/api/audio/upload"

payload = {
    "latitude": "-6.917464",
    "longitude": "107.619123"
}

with open(FILENAME, "rb") as f:
    files = {"audio": (FILENAME, f, "audio/wav")}
    response = requests.post(url, data=payload, files=files)

#Tampilkan Hasil Respon dari Server
if response.status_code == 200:
    print("\nSUKSES! Respon Server Go:")
    print(response.json())
else:
    print(f"\nGAGAL ({response.status_code}):", response.text)