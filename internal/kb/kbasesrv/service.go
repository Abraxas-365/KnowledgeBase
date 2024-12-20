package kbsrv

import (
	"context"
	"fmt"
	"time"

	"github.com/Abraxas-365/opd/internal/chatuser/chatusersrv"
	"github.com/Abraxas-365/opd/internal/interaction"
	"github.com/Abraxas-365/opd/internal/interaction/interactionsrv"
	"github.com/Abraxas-365/opd/internal/kb"
	"github.com/Abraxas-365/opd/internal/user/usersrv"
	"github.com/Abraxas-365/toolkit/pkg/database"
	"github.com/Abraxas-365/toolkit/pkg/errors"
	"github.com/Abraxas-365/toolkit/pkg/s3client"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockagentruntime/types"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/service/bedrockagent"
	"github.com/google/uuid"
)

type Service struct {
	kbClient           *bedrockagentruntime.Client
	brClient           *bedrockagent.BedrockAgent
	repo               kb.Repository
	userService        usersrv.Service
	userChatService    chatusersrv.Service
	interactionService interactionsrv.Service
	s3Client           s3client.Client
}

func New(kbClient *bedrockagentruntime.Client,
	brClient *bedrockagent.BedrockAgent,
	repo kb.Repository,
	s3 s3client.Client,
	userService usersrv.Service,
	userChatService chatusersrv.Service,
	InteractionService interactionsrv.Service,
) *Service {
	return &Service{
		kbClient:           kbClient,
		repo:               repo,
		brClient:           brClient,
		s3Client:           s3,
		userService:        userService,
		userChatService:    userChatService,
		interactionService: InteractionService,
	}
}

func (s *Service) CompleteAnswerWithMetadata(ctx context.Context, userMessage string, sessionID *string, userchatID string) (*bedrockagentruntime.RetrieveAndGenerateOutput, error) {
	kbConf, err := s.repo.GetKnowlegeBaseConfig()
	if err != nil {
		return nil, err
	}

	if _, err := s.userChatService.GetChatUserByID(ctx, userchatID); err != nil {
		return nil, err
	}

	output, err := s.kbClient.RetrieveAndGenerate(
		context.TODO(),
		&bedrockagentruntime.RetrieveAndGenerateInput{
			SessionId: sessionID,
			Input: &types.RetrieveAndGenerateInput{
				Text: aws.String(userMessage),
			},
			RetrieveAndGenerateConfiguration: &types.RetrieveAndGenerateConfiguration{
				Type: types.RetrieveAndGenerateTypeKnowledgeBase,
				KnowledgeBaseConfiguration: &types.KnowledgeBaseRetrieveAndGenerateConfiguration{
					KnowledgeBaseId: aws.String(kbConf.ID),
					ModelArn:        aws.String(kbConf.Model.ModelId),
					RetrievalConfiguration: &types.KnowledgeBaseRetrievalConfiguration{
						VectorSearchConfiguration: &types.KnowledgeBaseVectorSearchConfiguration{
							NumberOfResults:    aws.Int32(int32(kbConf.NumberOfResults)),
							OverrideSearchType: types.SearchTypeHybrid,
						},
					},
					GenerationConfiguration: &types.GenerationConfiguration{
						PromptTemplate: &types.PromptTemplate{
							TextPromptTemplate: aws.String(kbConf.Model.Prompt),
						},
						InferenceConfig: &types.InferenceConfig{
							TextInferenceConfig: &types.TextInferenceConfig{
								Temperature:   aws.Float32(0),
								TopP:          aws.Float32(1),
								MaxTokens:     aws.Int32(2048),
								StopSequences: []string{"\nObservation"},
							},
						},
					},
					OrchestrationConfiguration: &types.OrchestrationConfiguration{
						QueryTransformationConfiguration: &types.QueryTransformationConfiguration{
							Type: types.QueryTransformationTypeQueryDecomposition,
						},
						InferenceConfig: &types.InferenceConfig{
							TextInferenceConfig: &types.TextInferenceConfig{
								Temperature:   aws.Float32(0),
								TopP:          aws.Float32(1),
								MaxTokens:     aws.Int32(2048),
								StopSequences: []string{"\nObservation"},
							},
						},
					},
				},
			},
		},
	)
	if err != nil {
		return nil, errors.ErrServiceUnavailable(err.Error())
	}

	var chatInformationContext []string
	for _, citation := range output.Citations {
		for _, ref := range citation.RetrievedReferences {
			if ref.Location == nil {
				continue
			}

			if ref.Location.S3Location == nil {
				continue
			}

			if ref.Location.S3Location.Uri == nil {
				continue
			}

			chatInformationContext = append(chatInformationContext, *ref.Location.S3Location.Uri)
		}
	}

	i := interaction.Interaction{
		UserChatID:         userchatID,
		ContextInteraction: chatInformationContext,
	}

	if _, err := s.interactionService.CreateInteraction(ctx, i); err != nil {
		return nil, err
	}

	return output, nil
}

func (s *Service) GeneratePutURL(userID string, file string) (string, error) {
	u, err := s.userService.GetUser(context.Background(), userID)
	if err != nil {
		return "", err
	}
	key := fmt.Sprintf("data/%s-%s", file, uuid.New().String())
	dataFile := kb.DataFile{
		Filename:  file,
		S3Key:     key,
		UserID:    userID,
		UserEmail: u.Email,
	}
	_, err = s.repo.SaveData(context.Background(), dataFile)
	if err != nil {
		return "", err
	}

	return s.s3Client.GeneratePresignedPutURL(key, 60*time.Second)
}

func (s *Service) GetFiles(ctx context.Context, page, pageSize int) (database.PaginatedRecord[kb.DataFile], error) {
	return s.repo.GetData(ctx, page, pageSize)
}

func (s *Service) DeleteObject(fileID int) error {
	file, err := s.repo.GetDataById(context.Background(), fileID)
	if err != nil {
		return err
	}
	if _, err := s.repo.DeleteData(context.Background(), fileID); err != nil {
		return err
	}

	return s.s3Client.DeleteFile(file.S3Key)

}

func (s *Service) LisObjects(pageSize int32, continuationToken *string) ([]string, *string, error) {
	return s.s3Client.ListFiles(pageSize, continuationToken)
}

func (s *Service) SyncKnowledgeBase(ctx context.Context) (*bedrockagent.StartIngestionJobOutput, error) {
	kbConf, err := s.repo.GetKnowlegeBaseConfig()
	if err != nil {
		return nil, err
	}
	// Set up the input for the StartIngestionJob API call
	input := &bedrockagent.StartIngestionJobInput{
		KnowledgeBaseId: aws.String(kbConf.ID),
		DataSourceId:    aws.String(kbConf.S3DataSurce),
	}

	// Call StartIngestionJob
	output, err := s.brClient.StartIngestionJob(input)
	if err != nil {
		// Use runtime type assertion with awserr.Error to get more details about the error
		if awsErr, ok := err.(awserr.Error); ok {
			switch awsErr.Code() {
			case bedrockagent.ErrCodeThrottlingException:
				return nil, errors.ErrServiceUnavailable("throttling error: " + awsErr.Message())
			case bedrockagent.ErrCodeAccessDeniedException:
				return nil, errors.ErrForbidden("access denied: " + awsErr.Message())
			case bedrockagent.ErrCodeValidationException:
				return nil, errors.ErrBadRequest("validation error: " + awsErr.Message())
			case bedrockagent.ErrCodeInternalServerException:
				return nil, errors.ErrUnexpected("internal server error: " + awsErr.Message())
			case bedrockagent.ErrCodeResourceNotFoundException:
				return nil, errors.ErrNotFound("resource not found: " + awsErr.Message())
			case bedrockagent.ErrCodeConflictException:
				return nil, errors.ErrConflict("conflict error: " + awsErr.Message())
			case bedrockagent.ErrCodeServiceQuotaExceededException:
				return nil, errors.ErrServiceUnavailable("service quota exceeded: " + awsErr.Message())
			default:
				return nil, errors.ErrUnexpected("unknown error: " + awsErr.Message())
			}
		}
		// For non-AWS specific errors
		return nil, errors.ErrServiceUnavailable(err.Error())
	}

	// Return the IngestionJobId to track the job status later
	return output, nil
}
