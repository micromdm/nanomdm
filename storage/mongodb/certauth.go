package mongodb

import (
	"context"
	"errors"

	"github.com/micromdm/nanomdm/mdm"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type CertEnrollment struct {
	EnrollmentID       string   `bson:"enrollment_id"`
	CertHash           string   `bson:"cert_hash,omitempty"`
	PreviousCertHashes []string `bson:"previous_cert_hashes,omitempty"`
}

// Checks if the specific hash cert exists anywhere
func (m MongoDBStorage) HasCertHash(r *mdm.Request, hash string) (bool, error) {
	filter := bson.M{
		"cert_hash": hash,
	}

	res := CertEnrollment{}
	err := m.CertAuthCollection.FindOne(context.TODO(), filter).Decode(&res)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, err
	}

	// found a document which matches the cert hash, contents does not matter
	return true, nil
}

// Checks if an enrollment any cert hash
func (m MongoDBStorage) EnrollmentHasCertHash(r *mdm.Request, hash string) (bool, error) {
	filter := bson.M{
		"cert_hash":     hash,
		"enrollment_id": r.ID,
	}

	res := CertEnrollment{}
	err := m.CertAuthCollection.FindOne(context.TODO(), filter).Decode(&res)
	if err != nil {
		if errors.Is(err, mongo.ErrNoDocuments) {
			return false, nil
		}
		return false, err
	}

	// found a document which matches the cert hash, contents does not matter
	return true, nil
}

// Checks if the cert hash matches the one associated with the enrollment
func (m MongoDBStorage) IsCertHashAssociated(r *mdm.Request, hash string) (bool, error) {
	filter := bson.M{
		"enrollment_id": r.ID,
	}
	res := CertEnrollment{}
	err := m.CertAuthCollection.FindOne(context.TODO(), filter).Decode(&res)
	if err != nil {
		return false, err
	}
	return hash == res.CertHash, nil
}

// Associate the cert hash with the requested enrollment
func (m MongoDBStorage) AssociateCertHash(r *mdm.Request, hash string) error {
	upsert := true
	filter := bson.M{
		"enrollment_id": r.ID,
	}
	update := bson.M{
		"$set": CertEnrollment{
			EnrollmentID: r.ID,
			CertHash:     hash,
		},
		"$push": bson.M{
			"previous_cert_hashes": hash,
		},
	}
	_, err := m.CertAuthCollection.UpdateOne(context.TODO(), filter, update, &options.UpdateOptions{Upsert: &upsert})
	if err != nil {
		return err
	}

	return nil
}
