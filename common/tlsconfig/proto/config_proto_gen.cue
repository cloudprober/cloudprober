package proto

#TLSConfig: {
	// CA certificate file to verify certificates provided by the other party.
	caCertFile?: string @protobuf(1,string,name=ca_cert_file)

	// Local certificate file.
	tlsCertFile?: string @protobuf(2,string,name=tls_cert_file)

	// Private key file corresponding to the certificate above.
	tlsKeyFile?: string @protobuf(3,string,name=tls_key_file)

	// Whether to ignore the cert validation.
	disableCertValidation?: bool @protobuf(4,bool,name=disable_cert_validation)

	// ServerName override
	serverName?: string @protobuf(5,string,name=server_name)
}
