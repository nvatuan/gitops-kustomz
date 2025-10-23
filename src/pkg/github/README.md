```
git clone --filter=blob:none --depth 1 --no-checkout $REPO sparseblobless/
git sparse-checkout set --no-cone services/app-bootstrap-touya

git checkout master #< head branch

git pull origin nvatuan-patch-9
git checkout -b nvatuan-patch-9


git sparse-checkout set --no-cone /services/app-bootstrap-touya
```

git clone -n --depth=1 --filter=tree:0 -b master --single-branch $REPO master2

sparse checkout