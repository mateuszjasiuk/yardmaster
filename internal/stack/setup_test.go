package stack

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestRepositorySettingsArgs(t *testing.T) {
	if got, want := repoSettingsArgs("mateuszjasiuk/yardmaster"), []string{"api", "repos/mateuszjasiuk/yardmaster"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("settings args = %v, want %v", got, want)
	}
	if got, want := enableAutoDeleteArgs("mateuszjasiuk/yardmaster"), []string{"api", "--method", "PATCH", "repos/mateuszjasiuk/yardmaster", "-F", "delete_branch_on_merge=true"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("enable args = %v, want %v", got, want)
	}
}

func TestDecodeRepositorySettings(t *testing.T) {
	var got repositoryResponse
	data := []byte(`{"full_name":"mateuszjasiuk/yardmaster","delete_branch_on_merge":true}`)
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got.FullName != "mateuszjasiuk/yardmaster" || !got.DeleteBranchOnMerge {
		t.Fatalf("decoded settings = %+v", got)
	}
}
