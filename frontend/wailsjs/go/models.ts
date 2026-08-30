export namespace services {
	
	export class FileInfo {
	    path: string;
	    name: string;
	    content: string;
	    size: number;
	    modTime: string;
	    isReadOnly: boolean;
	
	    static createFrom(source: any = {}) {
	        return new FileInfo(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.content = source["content"];
	        this.size = source["size"];
	        this.modTime = source["modTime"];
	        this.isReadOnly = source["isReadOnly"];
	    }
	}
	export class TextStats {
	    lines: number;
	    words: number;
	    characters: number;
	    readingTime: string;
	
	    static createFrom(source: any = {}) {
	        return new TextStats(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.lines = source["lines"];
	        this.words = source["words"];
	        this.characters = source["characters"];
	        this.readingTime = source["readingTime"];
	    }
	}

}

