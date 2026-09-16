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

