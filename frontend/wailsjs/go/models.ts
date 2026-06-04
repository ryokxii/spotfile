export namespace engine {
	
	export class SearchResult {
	    docPath: string;
	    chunkIdx: number;
	    text: string;
	    score: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.docPath = source["docPath"];
	        this.chunkIdx = source["chunkIdx"];
	        this.text = source["text"];
	        this.score = source["score"];
	    }
	}

}

export namespace main {
	
	export class EngineConfig {
	    libraryPath: string;
	    modelPath: string;
	    vocabPath: string;
	    workers: number;
	
	    static createFrom(source: any = {}) {
	        return new EngineConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.libraryPath = source["libraryPath"];
	        this.modelPath = source["modelPath"];
	        this.vocabPath = source["vocabPath"];
	        this.workers = source["workers"];
	    }
	}

}

