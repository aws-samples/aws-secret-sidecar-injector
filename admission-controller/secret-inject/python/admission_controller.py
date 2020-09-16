import base64
import json
import os
import boto3
from flask import jsonify, Flask, request

app = Flask(__name__)

@app.route("/mutating-pods", methods=["POST"])
def mutation():
    review = request.get_json()
    #app.logger.info("Mutating AdmissionReview request: %s", json.dumps(review, indent=4))

    annotations = review['request']['object']['metadata']['annotations']
    app.logger.info("Annotations on the pod are: %s",annotations)
    
    response = {}

    # Only allow if there are valid annotationn
    if 'secrets.k8s.aws/sidecarInjectorWebhook' and 'secrets.k8s.aws/secret-arn' not in list(annotations):
        app.logger.info("Nothing to do because of missing annotations ...")
    else:
        app.logger.info("Annotations present ...")
        app.logger.info("Injecting init container to the pod definition ...")
        response = secrets_initcont_patch(annotations,response)    
    
    response['allowed'] = True
    review['response'] = response
    #app.logger.info("Mutating AdmissionReview request: %s", review)
    return jsonify(review), 200

# Prepare the patch and compose the response json
def secrets_initcont_patch(annotations,response):

    patch = [
        {
        	"op": "add",
        	"path": "/spec/initContainers",
        	"value": [
                {
        		    "image": "%v",
        		    "name": "secrets-init-container",
		            "volumeMounts": [
                        {
		            	    "name": "secret-vol",
		            	    "mountPath": "/tmp"
		                }
                    ],
        		    "env": [
                        {
        		    	    "name": "SECRET_ARN",
        		    	    "valueFrom": {
        		    	    	"fieldRef": {
        		    	    		"fieldPath": "metadata.annotations['secrets.k8s.aws/secret-arn']"
        		    	    	}
        		    	    }
        		        }
                    ],
        		    "resources": {}
        	    }
            ]
        },
        {
	        "op": "add",
	        "path": "/spec/volumes/-",
	        "value": 
            {
	        	"emptyDir": 
                {
	        		"medium": "Memory"
	        	},
	        	"name": "secret-vol"
	        }
        }
    ]

    response['patch'] = base64.b64encode(json.dumps(patch))
    response['patchType'] = 'application/json-patch+json'

    return response

# Enabling TLS for the flask application
context = (
    os.environ.get("WEBHOOK_CERT", "/tls/tls.crt"),
    os.environ.get("WEBHOOK_KEY", "/tls/tls.key"),
)
app.run(host='0.0.0.0', port='443', debug=False, ssl_context=context)