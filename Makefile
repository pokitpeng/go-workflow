rsync:
	rsync -avz -e "ssh -p 7022" --delete --exclude={\*.git,\*rpm,.vscode,.idea,node_modules,.venv} `pwd` root@117.50.85.144:/root