"use client";

import { useState } from "react";
import Image from "next/image";


export default function Home() {
  const [file, setFile] = useState<File | null>(null);
  const [isDragging, setIsDragging] = useState(false);
  const [isUploading, setIsUploading] = useState(false);
  const [uploadSuccess, setUploadSuccess] = useState(false);
  const [usingTestFile, setUsingTestFile] = useState(false);

  const handleFileChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    if (e.target.files && e.target.files[0]) {
      setFile(e.target.files[0]);
      setUploadSuccess(false);
    }
  };

  const handleDragOver = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(true);
  };

  const handleDragLeave = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);
  };

  const handleDrop = (e: React.DragEvent) => {
    e.preventDefault();
    setIsDragging(false);

    if (e.dataTransfer.files && e.dataTransfer.files[0]) {
      const droppedFile = e.dataTransfer.files[0];
      if (droppedFile.type === "text/csv" || droppedFile.name.endsWith(".csv")) {
        setFile(droppedFile);
        setUploadSuccess(false);
      } else {
        alert("Please upload a CSV file");
      }
    }
  };

  const handleUseTestFile = async () => {
    setIsUploading(true);
    setUsingTestFile(true);
    
    try {
      // Send a request to use the test.csv file that's already on the server
      const response = await fetch('/api/use-test-file', {
        method: 'POST',
      });
      
      if (response.ok) {
        setUploadSuccess(true);
      } else {
        throw new Error('Failed to use test file');
      }
    } catch (error) {
      console.error('Error using test file:', error);
      alert('Failed to use test file. Please try again.');
    } finally {
      setIsUploading(false);
    }
  };
  
  const handleSubmit = async () => {
    if (!file) return;

    setIsUploading(true);
    setUsingTestFile(false);

    try {
      const formData = new FormData();
      formData.append('file', file);

      // Replace with your actual API endpoint
      const response = await fetch('/api/upload', {
        method: 'POST',
        body: formData,
      });

      if (response.ok) {
        setUploadSuccess(true);
        setFile(null);
      } else {
        throw new Error('Upload failed');
      }
    } catch (error) {
      console.error('Error uploading file:', error);
      alert('Failed to upload file. Please try again.');
    } finally {
      setIsUploading(false);
    }
  };

  return (
    <main className="flex h-auto flex-col items-center p-8 md:p-12">
      <div className="mb-10">
        <h1 className="text-4xl font-bold text-center text-stone-700">Query Your Data</h1>
      </div>

      <div className="w-full max-w-2xl mt-6">
        <div
          className={`border-2 border-dashed rounded-lg p-8 flex flex-col items-center justify-center transition-colors ${
            isDragging
              ? "border-blue-500 bg-blue-50"
              : uploadSuccess
                ? "border-green-500 bg-green-50"
                : "border-gray-300 bg-gray-50 hover:bg-gray-100"
          }`}
          onDragOver={handleDragOver}
          onDragLeave={handleDragLeave}
          onDrop={handleDrop}
        >
          <div className="mb-5">
            <svg
              className={`w-16 h-16 ${
                uploadSuccess
                  ? "text-green-500"
                  : "text-gray-400"
              }`}
              fill="none"
              stroke="currentColor"
              viewBox="0 0 24 24"
              xmlns="http://www.w3.org/2000/svg"
            >
              <path
                strokeLinecap="round"
                strokeLinejoin="round"
                strokeWidth={1.5}
                d="M9 17v-2m3 2v-4m3 4v-6m2 10H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"
              />
            </svg>
          </div>

          {uploadSuccess ? (
            <div className="text-center">
              <div className="text-lg font-medium text-green-600 mb-2">Upload Successful!</div>
              <p className="text-sm text-green-500 mb-4">Your file has been uploaded successfully.</p>
              <button
                className="px-4 py-2 bg-stone-700 text-white rounded-md hover:bg-stone-800 transition-colors"
                onClick={() => setUploadSuccess(false)}
              >
                Upload Another File
              </button>
            </div>
          ) : (
            <>
              <div className="text-lg font-medium text-gray-700 mb-2">
                {file ? file.name : "Drop your CSV file here"}
              </div>
              <p className="text-sm text-gray-500 mb-4">
                {file ? `${(file.size / 1024).toFixed(2)} KB` : "or click to browse"}
              </p>

              <div className="flex flex-col sm:flex-row gap-4 w-full max-w-xs">
                <label className="flex-1">
                  <input
                    type="file"
                    className="hidden"
                    accept=".csv"
                    onChange={handleFileChange}
                  />
                  <div className="w-full px-4 py-2 text-center text-gray-600 bg-white border border-gray-300 rounded-md cursor-pointer hover:bg-gray-50 transition-colors">
                    Browse Files
                  </div>
                </label>

                {file && (
                  <button
                    className={`flex-1 px-4 py-2 text-white rounded-md transition-colors ${
                      isUploading
                        ? "bg-stone-500 cursor-not-allowed"
                        : "bg-stone-700 hover:bg-stone-800"
                    }`}
                    onClick={handleSubmit}
                    disabled={isUploading}
                  >
                    {isUploading ? "Uploading..." : "Upload"}
                  </button>
                )}
              </div>
              
              <div className="text-center mt-3">
                <div className="text-sm text-gray-500 mb-1">or</div>
                <button
                  onClick={handleUseTestFile}
                  className="text-blue-600 hover:text-blue-800 transition-colors font-medium underline"
                >
                  Use sample test.csv instead
                </button>
              </div>
            </>
          )}
        </div>

        <div className="mt-4 text-sm text-gray-500 text-center">
          Supported format: CSV files only
        </div>
      </div>
    </main>
  );
}


