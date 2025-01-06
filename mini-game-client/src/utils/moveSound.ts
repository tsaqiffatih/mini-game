export default function playMoveSound(): void {
    const moveSound = new Audio("/sounds/Move.mp3");
    const playMove = moveSound.play();
    
    if (playMove !== undefined) {
        playMove
            .then(() => {})
            .catch((error: Error) => {
                if (process.env.NODE_ENV === 'development') {
                    console.log("Failed to load move sound", error);
                }
            });
    }
}