SQLITE_WSD struct Sqlite3Config sqlite3Config = {
  SQLITE_DEFAULT_MEMSTATUS,  /* bMemstat */
  1,                         /* bCoreMutex */
  SQLITE_THREADSAFE==1,      /* bFullMutex */
  SQLITE_USE_URI,            /* bOpenUri */
  SQLITE_ALLOW_COVERING_INDEX_SCAN,   /* bUseCis */
  0,                         /* bSmallMalloc */
  1,                         /* bExtraSchemaChecks */
#ifdef SQLITE_DEBUG
  0,                         /* bJsonSelfcheck */
#endif
  0x7ffffffe,                /* mxStrlen */
  0,                         /* neverCorrupt */
  SQLITE_DEFAULT_LOOKASIDE,  /* szLookaside, nLookaside */
  SQLITE_STMTJRNL_SPILL,     /* nStmtSpill */
  {0,0,0,0,0,0,0,0},         /* m */
  {0,0,0,0,0,0,0,0,0},       /* mutex */
  {0,0,0,0,0,0,0,0,0,0,0,0,0},/* pcache2 */
  (void*)0,                  /* pHeap */
  0,                         /* nHeap */
  0, 0,                      /* mnHeap, mxHeap */
  SQLITE_DEFAULT_MMAP_SIZE,  /* szMmap */
  SQLITE_MAX_MMAP_SIZE,      /* mxMmap */
  (void*)0,                  /* pPage */
  0,                         /* szPage */
  SQLITE_DEFAULT_PCACHE_INITSZ, /* nPage */
  0,                         /* mxParserStack */
  0,                         /* sharedCacheEnabled */
  SQLITE_SORTER_PMASZ,       /* szPma */
  /* All the rest should always be initialized to zero */
  0,                         /* isInit */
  0,                         /* inProgress */
  0,                         /* isMutexInit */
  0,                         /* isMallocInit */
  0,                         /* isPCacheInit */
  0,                         /* nRefInitMutex */
  0,                         /* pInitMutex */
  0,                         /* xLog */
  0,                         /* pLogArg */
#ifdef SQLITE_ENABLE_SQLLOG
  0,                         /* xSqllog */
  0,                         /* pSqllogArg */
#endif
#ifdef SQLITE_VDBE_COVERAGE
  0,                         /* xVdbeBranch */
  0,                         /* pVbeBranchArg */
#endif
#ifndef SQLITE_OMIT_DESERIALIZE
  SQLITE_MEMDB_DEFAULT_MAXSIZE,   /* mxMemdbSize */
#endif
#ifndef SQLITE_UNTESTABLE
  0,                         /* xTestCallback */
#endif
#ifdef SQLITE_ALLOW_ROWID_IN_VIEW
  0,                         /* mNoVisibleRowid.  0 == allow rowid-in-view */
#endif
  0,                         /* bLocaltimeFault */
  0,                         /* xAltLocaltime */
  0x7ffffffe,                /* iOnceResetThreshold */
  SQLITE_DEFAULT_SORTERREF_SIZE,   /* szSorterRef */
  0,                         /* iPrngSeed */
#ifdef SQLITE_DEBUG
  {0,0,0,0,0,0},             /* aTune */
#endif
};