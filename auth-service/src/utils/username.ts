import prisma from "../libs/prisma";

/**
 * Generates a unique username from a base string (name or email)
 * @param base - The base string to generate username from
 * @returns A unique username
 */
export async function generateUniqueUsername(base: string): Promise<string> {
    // Clean the base string: lowercase, remove special chars, replace spaces with underscores
    let username = base
        .toLowerCase()
        .replace(/[^a-z0-9_]/g, "")
        .replace(/\s+/g, "_")
        .substring(0, 30); // Limit length
    
    // Remove leading/trailing underscores
    username = username.replace(/^_+|_+$/g, "");
    
    // If empty after cleaning, use a default
    if (!username) {
        username = "user";
    }
    
    // Check if username is available
    let candidate = username;
    let counter = 1;
    const maxAttempts = 1000;
    
    while (counter < maxAttempts) {
        const exists = await prisma.user.findUnique({
            where: { username: candidate },
            select: { id: true }
        });
        
        if (!exists) {
            return candidate;
        }
        
        // Try with numbers
        candidate = `${username}${counter}`;
        counter++;
    }
    
    // Fallback: use timestamp if all attempts fail
    return `${username}_${Date.now()}`;
}

/**
 * Generates available username suggestions based on the requested username
 * @param requestedUsername - The username that was already taken
 * @param maxSuggestions - Maximum number of suggestions to generate (default: 3)
 * @returns Array of available username suggestions
 */
export async function generateUsernameSuggestions(
    requestedUsername: string,
    maxSuggestions: number = 3
): Promise<string[]> {
    const suggestions: string[] = [];
    const attempts = 50; // Max attempts to find available usernames
    
    // Generate variations
    const variations: (() => string)[] = [
        // Add numbers
        () => `${requestedUsername}${Math.floor(Math.random() * 1000)}`,
        () => `${requestedUsername}${Math.floor(Math.random() * 10000)}`,
        () => `${requestedUsername}_${Math.floor(Math.random() * 100)}`,
        
        // Add common suffixes
        () => `${requestedUsername}123`,
        () => `${requestedUsername}2024`,
        () => `${requestedUsername}_user`,
        
        // Add random strings
        () => `${requestedUsername}_${Math.random().toString(36).substring(2, 6)}`,
        () => `${requestedUsername}_${Math.random().toString(36).substring(2, 8)}`,
        
        // Variations with underscores
        () => `${requestedUsername}_${Math.floor(Math.random() * 999)}`,
        () => `${requestedUsername}${Math.floor(Math.random() * 999)}`,
        
        // More random variations
        () => `${requestedUsername}${Math.random().toString(36).substring(2, 5)}`,
        () => `${requestedUsername}_${Date.now().toString().slice(-6)}`,
    ];
    
    let attemptCount = 0;
    
    while (suggestions.length < maxSuggestions && attemptCount < attempts) {
        // Try different variation strategies
        const variationIndex = attemptCount % variations.length;
        const candidate = variations[variationIndex]();
        
        // Ensure candidate is valid (non-empty, reasonable length)
        if (candidate.length > 0 && candidate.length <= 50) {
            // Check if username is available
            const exists = await prisma.user.findUnique({
                where: { username: candidate },
                select: { id: true }
            });
            
            if (!exists && !suggestions.includes(candidate)) {
                suggestions.push(candidate);
            }
        }
        
        attemptCount++;
    }
    
    return suggestions;
}

