export namespace model {
	
	export class AppConfig {
	    projectA: string;
	    projectB: string;
	    direction: string;
	    windowWidth: number;
	    windowHeight: number;
	    aiCommitApiKey?: string;
	
	    static createFrom(source: any = {}) {
	        return new AppConfig(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.projectA = source["projectA"];
	        this.projectB = source["projectB"];
	        this.direction = source["direction"];
	        this.windowWidth = source["windowWidth"];
	        this.windowHeight = source["windowHeight"];
	        this.aiCommitApiKey = source["aiCommitApiKey"];
	    }
	}
	export class TargetRepositoryStatus {
	    path: string;
	    name: string;
	    branch: string;
	    isGitRepo: boolean;
	    error: string;
	    isClean: boolean;
	    modifiedCount: number;
	    untrackedCount: number;
	    canPush: boolean;
	
	    static createFrom(source: any = {}) {
	        return new TargetRepositoryStatus(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.name = source["name"];
	        this.branch = source["branch"];
	        this.isGitRepo = source["isGitRepo"];
	        this.error = source["error"];
	        this.isClean = source["isClean"];
	        this.modifiedCount = source["modifiedCount"];
	        this.untrackedCount = source["untrackedCount"];
	        this.canPush = source["canPush"];
	    }
	}
	export class DiffSummary {
	    total: number;
	    added: number;
	    modified: number;
	    deleted: number;
	
	    static createFrom(source: any = {}) {
	        return new DiffSummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.total = source["total"];
	        this.added = source["added"];
	        this.modified = source["modified"];
	        this.deleted = source["deleted"];
	    }
	}
	export class DiffEntry {
	    path: string;
	    kind: string;
	    sizeBytes: number;
	
	    static createFrom(source: any = {}) {
	        return new DiffEntry(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.path = source["path"];
	        this.kind = source["kind"];
	        this.sizeBytes = source["sizeBytes"];
	    }
	}
	export class RepositorySummary {
	    slot: string;
	    path: string;
	    name: string;
	    isConfigured: boolean;
	    isGitRepo: boolean;
	    validationError: string;
	    branch: string;
	    isClean: boolean;
	    modifiedCount: number;
	    untrackedCount: number;
	
	    static createFrom(source: any = {}) {
	        return new RepositorySummary(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.slot = source["slot"];
	        this.path = source["path"];
	        this.name = source["name"];
	        this.isConfigured = source["isConfigured"];
	        this.isGitRepo = source["isGitRepo"];
	        this.validationError = source["validationError"];
	        this.branch = source["branch"];
	        this.isClean = source["isClean"];
	        this.modifiedCount = source["modifiedCount"];
	        this.untrackedCount = source["untrackedCount"];
	    }
	}
	export class DashboardState {
	    config: AppConfig;
	    aiCommitConfigured: boolean;
	    repositoryA: RepositorySummary;
	    repositoryB: RepositorySummary;
	    sourceSlot: string;
	    targetSlot: string;
	    differences: DiffEntry[];
	    summary: DiffSummary;
	    targetStatus: TargetRepositoryStatus;
	    canSync: boolean;
	
	    static createFrom(source: any = {}) {
	        return new DashboardState(source);
	    }
	
	    constructor(source: any = {}) {
	        if ('string' === typeof source) source = JSON.parse(source);
	        this.config = this.convertValues(source["config"], AppConfig);
	        this.aiCommitConfigured = source["aiCommitConfigured"];
	        this.repositoryA = this.convertValues(source["repositoryA"], RepositorySummary);
	        this.repositoryB = this.convertValues(source["repositoryB"], RepositorySummary);
	        this.sourceSlot = source["sourceSlot"];
	        this.targetSlot = source["targetSlot"];
	        this.differences = this.convertValues(source["differences"], DiffEntry);
	        this.summary = this.convertValues(source["summary"], DiffSummary);
	        this.targetStatus = this.convertValues(source["targetStatus"], TargetRepositoryStatus);
	        this.canSync = source["canSync"];
	    }
	
		convertValues(a: any, classs: any, asMap: boolean = false): any {
		    if (!a) {
		        return a;
		    }
		    if (a.slice && a.map) {
		        return (a as any[]).map(elem => this.convertValues(elem, classs));
		    } else if ("object" === typeof a) {
		        if (asMap) {
		            for (const key of Object.keys(a)) {
		                a[key] = new classs(a[key]);
		            }
		            return a;
		        }
		        return new classs(a);
		    }
		    return a;
		}
	}
	
	
	

}

