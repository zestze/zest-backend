'use client'

import { useState, useEffect } from "react";


const BACKEND_URL = 'http://localhost:8080'

export interface Post {
    title: string;
    href: string;
    score: number;
    description: string;
    release_date: Date | string;
}

// TODO(zeke): use SWR since that's what's recommended by the nextjs devs
export const usePosts = (medium: string, minYear: number, maxYear: number): [Post[], boolean] => {
    const [posts, setPosts] = useState<Post[]>([] as Post[]);
    const [isLoading, setLoading] = useState<boolean>(true);
    useEffect(() => {
        fetch(`${BACKEND_URL}/v1/posts?medium=${medium}&min_year=${minYear}&max_year=${maxYear}`)
            .then(response => response.json())
            .then(data => {
                setPosts(toPosts(data.posts));
                setLoading(false);
            })
            .catch(error => console.error(error));
    }, [medium, minYear, maxYear])
    return [posts, isLoading]
};

const toPosts = (data: Post[]): Post[] => {
    return data.map((post: Post): Post => {
        return {
            ...post,
            release_date: new Date(post.release_date as string)
        }
    });
}