"use client";

<<<<<<< HEAD

import { useState } from "react";


export default function Header() {
    const [sessionActive, setSessionActive] = useState(false);
   
    const toggleSession = () => {
        setSessionActive(!sessionActive);
    };
   
=======
import { useState } from "react";

export default function Header() {
    const [sessionActive, setSessionActive] = useState(false);
    
    const toggleSession = () => {
        setSessionActive(!sessionActive);
    };
    
>>>>>>> 392e8b9f8b13b41459dc975e4da1975f82adf3fa
    return (
        <header className="p-5">
            <div className="container mx-auto flex justify-between items-center">
                <h1 className="text-3xl font-bold text-stone-800">ASKQL</h1>
<<<<<<< HEAD
                  <div
                    className={`flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-full cursor-pointer transition-all duration-300 ${
                        sessionActive
                            ? "bg-red-100 text-red-600"
=======
                  <div 
                    className={`flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-full cursor-pointer transition-all duration-300 ${
                        sessionActive 
                            ? "bg-red-100 text-red-600" 
>>>>>>> 392e8b9f8b13b41459dc975e4da1975f82adf3fa
                            : "bg-gray-100 text-gray-500"
                    }`}
                    onClick={toggleSession}
                    >
                    <div className={`relative w-2.5 h-2.5 rounded-full ${
<<<<<<< HEAD
                        sessionActive
                            ? "bg-red-600"
=======
                        sessionActive 
                            ? "bg-red-600" 
>>>>>>> 392e8b9f8b13b41459dc975e4da1975f82adf3fa
                            : "bg-gray-400"
                    }`}>
                        {sessionActive && (
                            <span className="absolute inset-0 rounded-full animate-ping bg-red-400 opacity-75"></span>
                        )}
                    </div>                    
                    <span className="font-medium text-xs">
                        {sessionActive ? "Session Active" : "Session Inactive"}
                    </span>
                </div>
            </div>
        </header>
    );
}
