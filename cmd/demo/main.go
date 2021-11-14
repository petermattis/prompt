package main

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/petermattis/prompt"
)

func init() {
	sort.Strings(sqlKeywords)
}

func completer(text []rune, wordStart, wordEnd int) []string {
	word := strings.ToUpper(string(text[wordStart:wordEnd]))
	i := sort.Search(len(sqlKeywords), func(i int) bool {
		return sqlKeywords[i] >= word
	})
	if i >= len(sqlKeywords) {
		return nil
	}
	word += "\xff"
	j := sort.Search(len(sqlKeywords), func(i int) bool {
		return sqlKeywords[i] >= word
	})
	return sqlKeywords[i:j]
}

func inputFinished(text string) bool {
	text = strings.TrimSpace(text)
	return strings.HasSuffix(text, ";")
}

func main() {
	fmt.Printf(`# command line demo
# - multi-line input terminated by a trailing semicolon
# - standard navigation and editing commands
# - history browsing and search
# - kill ring
# - tab completion of SQL keywords
`)

	p := prompt.New(prompt.WithCompleter(completer),
		prompt.WithInputFinished(inputFinished))
	for {
		_, err := p.ReadLine("demo> ")
		if err != nil {
			log.Fatal(err)
		}
	}
}

// NB: copied from github.com/cockroachdb/cockroach/pkg/sql/lexbase/keywords.go:KeywordNames.
var sqlKeywords = []string{
	"ABORT",
	"ACCESS",
	"ACTION",
	"ADD",
	"ADMIN",
	"AFTER",
	"AGGREGATE",
	"ALL",
	"ALTER",
	"ALWAYS",
	"ANALYSE",
	"ANALYZE",
	"AND",
	"ANNOTATE_TYPE",
	"ANY",
	"ARRAY",
	"AS",
	"ASC",
	"ASYMMETRIC",
	"AT",
	"ATTRIBUTE",
	"AUTHORIZATION",
	"AUTOMATIC",
	"AVAILABILITY",
	"BACKUP",
	"BACKUPS",
	"BEFORE",
	"BEGIN",
	"BETWEEN",
	"BIGINT",
	"BINARY",
	"BIT",
	"BOOLEAN",
	"BOTH",
	"BOX2D",
	"BUCKET_COUNT",
	"BUNDLE",
	"BY",
	"CACHE",
	"CANCEL",
	"CANCELQUERY",
	"CASCADE",
	"CASE",
	"CAST",
	"CHANGEFEED",
	"CHAR",
	"CHARACTER",
	"CHARACTERISTICS",
	"CHECK",
	"CLOSE",
	"CLUSTER",
	"COALESCE",
	"COLLATE",
	"COLLATION",
	"COLUMN",
	"COLUMNS",
	"COMMENT",
	"COMMENTS",
	"COMMIT",
	"COMMITTED",
	"COMPACT",
	"COMPLETE",
	"CONCURRENTLY",
	"CONFIGURATION",
	"CONFIGURATIONS",
	"CONFIGURE",
	"CONFLICT",
	"CONNECTION",
	"CONSTRAINT",
	"CONSTRAINTS",
	"CONTROLCHANGEFEED",
	"CONTROLJOB",
	"CONVERSION",
	"CONVERT",
	"COPY",
	"COVERING",
	"CREATE",
	"CREATEDB",
	"CREATELOGIN",
	"CREATEROLE",
	"CROSS",
	"CSV",
	"CUBE",
	"CURRENT",
	"CURRENT_CATALOG",
	"CURRENT_DATE",
	"CURRENT_ROLE",
	"CURRENT_SCHEMA",
	"CURRENT_TIME",
	"CURRENT_TIMESTAMP",
	"CURRENT_USER",
	"CURSOR",
	"CYCLE",
	"DATA",
	"DATABASE",
	"DATABASES",
	"DAY",
	"DEALLOCATE",
	"DEBUG_PAUSE_ON",
	"DEC",
	"DECIMAL",
	"DECLARE",
	"DEFAULT",
	"DEFAULTS",
	"DEFERRABLE",
	"DEFERRED",
	"DELETE",
	"DELIMITER",
	"DESC",
	"DESTINATION",
	"DETACHED",
	"DISCARD",
	"DISTINCT",
	"DO",
	"DOMAIN",
	"DOUBLE",
	"DROP",
	"ELSE",
	"ENCODING",
	"ENCRYPTION_PASSPHRASE",
	"END",
	"ENUM",
	"ENUMS",
	"ESCAPE",
	"EXCEPT",
	"EXCLUDE",
	"EXCLUDING",
	"EXECUTE",
	"EXECUTION",
	"EXISTS",
	"EXPERIMENTAL",
	"EXPERIMENTAL_AUDIT",
	"EXPERIMENTAL_FINGERPRINTS",
	"EXPERIMENTAL_RELOCATE",
	"EXPERIMENTAL_REPLICA",
	"EXPIRATION",
	"EXPLAIN",
	"EXPORT",
	"EXTENSION",
	"EXTRACT",
	"EXTRACT_DURATION",
	"FAILURE",
	"FALSE",
	"FAMILY",
	"FETCH",
	"FILES",
	"FILTER",
	"FIRST",
	"FLOAT",
	"FOLLOWING",
	"FOR",
	"FORCE",
	"FORCE_INDEX",
	"FORCE_ZIGZAG",
	"FOREIGN",
	"FROM",
	"FULL",
	"FUNCTION",
	"FUNCTIONS",
	"GENERATED",
	"GEOGRAPHY",
	"GEOMETRY",
	"GEOMETRYCOLLECTION",
	"GEOMETRYCOLLECTIONM",
	"GEOMETRYCOLLECTIONZ",
	"GEOMETRYCOLLECTIONZM",
	"GEOMETRYM",
	"GEOMETRYZ",
	"GEOMETRYZM",
	"GLOBAL",
	"GOAL",
	"GRANT",
	"GRANTS",
	"GREATEST",
	"GROUP",
	"GROUPING",
	"GROUPS",
	"HASH",
	"HAVING",
	"HIGH",
	"HISTOGRAM",
	"HOUR",
	"IDENTITY",
	"IF",
	"IFERROR",
	"IFNULL",
	"IGNORE_FOREIGN_KEYS",
	"ILIKE",
	"IMMEDIATE",
	"IMPORT",
	"IN",
	"INCLUDE",
	"INCLUDING",
	"INCREMENT",
	"INCREMENTAL",
	"INDEX",
	"INDEXES",
	"INHERITS",
	"INITIALLY",
	"INJECT",
	"INNER",
	"INSERT",
	"INT",
	"INTEGER",
	"INTERSECT",
	"INTERVAL",
	"INTO",
	"INTO_DB",
	"INVERTED",
	"IS",
	"ISERROR",
	"ISNULL",
	"ISOLATION",
	"JOB",
	"JOBS",
	"JOIN",
	"JSON",
	"KEY",
	"KEYS",
	"KMS",
	"KV",
	"LANGUAGE",
	"LAST",
	"LATERAL",
	"LATEST",
	"LC_COLLATE",
	"LC_CTYPE",
	"LEADING",
	"LEASE",
	"LEAST",
	"LEFT",
	"LESS",
	"LEVEL",
	"LIKE",
	"LIMIT",
	"LINESTRING",
	"LINESTRINGM",
	"LINESTRINGZ",
	"LINESTRINGZM",
	"LIST",
	"LOCAL",
	"LOCALITY",
	"LOCALTIME",
	"LOCALTIMESTAMP",
	"LOCKED",
	"LOGIN",
	"LOOKUP",
	"LOW",
	"MATCH",
	"MATERIALIZED",
	"MAXVALUE",
	"MERGE",
	"METHOD",
	"MINUTE",
	"MINVALUE",
	"MODIFYCLUSTERSETTING",
	"MONTH",
	"MULTILINESTRING",
	"MULTILINESTRINGM",
	"MULTILINESTRINGZ",
	"MULTILINESTRINGZM",
	"MULTIPOINT",
	"MULTIPOINTM",
	"MULTIPOINTZ",
	"MULTIPOINTZM",
	"MULTIPOLYGON",
	"MULTIPOLYGONM",
	"MULTIPOLYGONZ",
	"MULTIPOLYGONZM",
	"NAMES",
	"NAN",
	"NATURAL",
	"NEVER",
	"NEW_DB_NAME",
	"NEXT",
	"NO",
	"NOCANCELQUERY",
	"NOCONTROLCHANGEFEED",
	"NOCONTROLJOB",
	"NOCREATEDB",
	"NOCREATELOGIN",
	"NOCREATEROLE",
	"NOLOGIN",
	"NOMODIFYCLUSTERSETTING",
	"NONE",
	"NON_VOTERS",
	"NORMAL",
	"NOT",
	"NOTHING",
	"NOTNULL",
	"NOVIEWACTIVITY",
	"NOWAIT",
	"NO_FULL_SCAN",
	"NO_INDEX_JOIN",
	"NO_ZIGZAG_JOIN",
	"NULL",
	"NULLIF",
	"NULLS",
	"NUMERIC",
	"OF",
	"OFF",
	"OFFSET",
	"OIDS",
	"ON",
	"ONLY",
	"OPERATOR",
	"OPT",
	"OPTION",
	"OPTIONS",
	"OR",
	"ORDER",
	"ORDINALITY",
	"OTHERS",
	"OUT",
	"OUTER",
	"OVER",
	"OVERLAPS",
	"OVERLAY",
	"OWNED",
	"OWNER",
	"PARENT",
	"PARTIAL",
	"PARTITION",
	"PARTITIONS",
	"PASSWORD",
	"PAUSE",
	"PAUSED",
	"PHYSICAL",
	"PLACEMENT",
	"PLACING",
	"PLAN",
	"PLANS",
	"POINT",
	"POINTM",
	"POINTZ",
	"POINTZM",
	"POLYGON",
	"POLYGONM",
	"POLYGONZ",
	"POLYGONZM",
	"POSITION",
	"PRECEDING",
	"PRECISION",
	"PREPARE",
	"PRESERVE",
	"PRIMARY",
	"PRIORITY",
	"PRIVILEGES",
	"PUBLIC",
	"PUBLICATION",
	"QUERIES",
	"QUERY",
	"RANGE",
	"RANGES",
	"READ",
	"REAL",
	"REASON",
	"REASSIGN",
	"RECURRING",
	"RECURSIVE",
	"REF",
	"REFERENCES",
	"REFRESH",
	"REGION",
	"REGIONAL",
	"REGIONS",
	"REINDEX",
	"RELEASE",
	"RENAME",
	"REPEATABLE",
	"REPLACE",
	"REPLICATION",
	"RESET",
	"RESTORE",
	"RESTRICT",
	"RESTRICTED",
	"RESUME",
	"RETRY",
	"RETURNING",
	"REVISION_HISTORY",
	"REVOKE",
	"RIGHT",
	"ROLE",
	"ROLES",
	"ROLLBACK",
	"ROLLUP",
	"ROUTINES",
	"ROW",
	"ROWS",
	"RULE",
	"RUNNING",
	"SAVEPOINT",
	"SCANS",
	"SCATTER",
	"SCHEDULE",
	"SCHEDULES",
	"SCHEMA",
	"SCHEMAS",
	"SCRUB",
	"SEARCH",
	"SECOND",
	"SELECT",
	"SEQUENCE",
	"SEQUENCES",
	"SERIALIZABLE",
	"SERVER",
	"SESSION",
	"SESSIONS",
	"SESSION_USER",
	"SET",
	"SETS",
	"SETTING",
	"SETTINGS",
	"SHARE",
	"SHOW",
	"SIMILAR",
	"SIMPLE",
	"SKIP",
	"SKIP_LOCALITIES_CHECK",
	"SKIP_MISSING_FOREIGN_KEYS",
	"SKIP_MISSING_SEQUENCES",
	"SKIP_MISSING_SEQUENCE_OWNERS",
	"SKIP_MISSING_VIEWS",
	"SMALLINT",
	"SNAPSHOT",
	"SOME",
	"SPLIT",
	"SQL",
	"START",
	"STATEMENTS",
	"STATISTICS",
	"STATUS",
	"STDIN",
	"STORAGE",
	"STORE",
	"STORED",
	"STORING",
	"STREAM",
	"STRICT",
	"STRING",
	"SUBSCRIPTION",
	"SUBSTRING",
	"SURVIVAL",
	"SURVIVE",
	"SYMMETRIC",
	"SYNTAX",
	"SYSTEM",
	"TABLE",
	"TABLES",
	"TABLESPACE",
	"TEMP",
	"TEMPLATE",
	"TEMPORARY",
	"TENANT",
	"TESTING_RELOCATE",
	"TEXT",
	"THEN",
	"THROTTLING",
	"TIES",
	"TIME",
	"TIMESTAMP",
	"TIMESTAMPTZ",
	"TIMETZ",
	"TO",
	"TRACE",
	"TRAILING",
	"TRANSACTION",
	"TRANSACTIONS",
	"TREAT",
	"TRIGGER",
	"TRIM",
	"TRUE",
	"TRUNCATE",
	"TRUSTED",
	"TYPE",
	"TYPES",
	"UNBOUNDED",
	"UNCOMMITTED",
	"UNION",
	"UNIQUE",
	"UNKNOWN",
	"UNLOGGED",
	"UNSPLIT",
	"UNTIL",
	"UPDATE",
	"UPSERT",
	"USE",
	"USER",
	"USERS",
	"USING",
	"VALID",
	"VALIDATE",
	"VALUE",
	"VALUES",
	"VARBIT",
	"VARCHAR",
	"VARIADIC",
	"VARYING",
	"VIEW",
	"VIEWACTIVITY",
	"VIRTUAL",
	"VISIBLE",
	"VOTERS",
	"WHEN",
	"WHERE",
	"WINDOW",
	"WITH",
	"WITHIN",
	"WITHOUT",
	"WORK",
	"WRITE",
	"YEAR",
	"ZONE",
}
