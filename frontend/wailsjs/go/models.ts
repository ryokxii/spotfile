export namespace main {
	
	export class EngineConfig {
	    modelPath: string;
	    serverPath: string;
	
	    static createFrom(source: any = {}) {
	        return new EngineConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.modelPath = source["modelPath"];
	        this.serverPath = source["serverPath"];
	    }
	}
	export class EngineStatus {
	    ready: boolean;
	    error: string;
	    documents: number;
	    chunks: number;
	
	    static createFrom(source: any = {}) {
	        return new EngineStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.ready = source["ready"];
	        this.error = source["error"];
	        this.documents = source["documents"];
	        this.chunks = source["chunks"];
	    }
	}

}

export namespace vectorstore {
	
	export class SearchResult {
	    docPath: string;
	    chunkIdx: number;
	    pageNum: number;
	    text: string;
	    score: number;
	
	    static createFrom(source: any = {}) {
	        return new SearchResult(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.docPath = source["docPath"];
	        this.chunkIdx = source["chunkIdx"];
	        this.pageNum = source["pageNum"];
	        this.text = source["text"];
	        this.score = source["score"];
	    }
	}

}

