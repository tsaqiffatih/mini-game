export default function playCaptureSound(): void {
    const captureSound = new Audio("./sounds/Capture.mp3");
    const playMove = captureSound.play();
    if (playMove !== undefined) {
        playMove
            .then(() => {})
            .catch((error) => {
                console.log("Failed to load capture sound", error);
            });
    }
}