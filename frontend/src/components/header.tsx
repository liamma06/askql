"use client";
import { useSession } from "@/contexts/SessionContext";

export default function Header() {
    const { sessionActive, sessionId, endSession } = useSession();
    return (
        <header className="p-5">
            <div className="container mx-auto flex justify-between items-center">
                <h1 className="text-3xl font-bold text-stone-800">ASKQL</h1>
                <div className="flex items-center gap-4">
                    {sessionActive && (
                        <button
                        onClick={endSession}
                        className="px-3 py-1.5 text-sm text-red-600 bg-red-50 rounded-md hover:bg-red-100 transition-colors"
                        >
                            End Session
                        </button>
                    )}
          
                    <div className={`flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-full transition-all duration-300 ${
                        sessionActive
                        ? "bg-red-100 text-red-600"
                        : "bg-gray-100 text-gray-500"
                    }`}>
                        <div className={`relative w-2.5 h-2.5 rounded-full
                        ${ sessionActive ? "bg-red-600" : "bg-gray-400"}`}>
                            {sessionActive && (
                                <span className="absolute inset-0 rounded-full animate-ping bg-red-400 opacity-75"></span>
                            )}
                        </div>
                        <span className="font-medium text-xs">
                            {sessionActive ? `Session Active${sessionId ? ` (${sessionId.slice(0, 8)}...)` : ''}` : "Session Inactive"}
                        </span>
                    </div>
                </div>
            </div>
        </header>
    );
}
