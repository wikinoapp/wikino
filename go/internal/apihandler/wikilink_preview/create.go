package wikilink_preview

import (
	"context"

	"github.com/wikinoapp/wikino/go/internal/apigen"
	apipage "github.com/wikinoapp/wikino/go/internal/apihandler/page"
	"github.com/wikinoapp/wikino/go/internal/middleware"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/usecase"
)

// CreateWikilinkPreviewは本文を書き込んだときに各Wikiリンクがどのページを指すかを、書き込まずに返す
// (POST /api/v1/spaces/{space_identifier}/wikilink_previews)。状態を変えない計算のため、200で返す
func (h *Handler) CreateWikilinkPreview(ctx context.Context, request apigen.CreateWikilinkPreviewRequestObject) (apigen.CreateWikilinkPreviewResponseObject, error) {
	principal := middleware.APIPrincipalFromContext(ctx)
	output, err := h.getAPIWikilinkPreviewUC.Execute(ctx, usecase.GetAPIWikilinkPreviewInput{
		Principal:       principal,
		SpaceIdentifier: model.SpaceIdentifier(request.SpaceIdentifier),
		TopicNumber:     request.Body.TopicNumber,
		Body:            request.Body.Body,
	})
	if err != nil {
		return nil, err
	}

	links := make([]apigen.WikilinkPreviewLink, 0, len(output.Links))
	for _, link := range output.Links {
		item := apigen.WikilinkPreviewLink{
			Text:       link.Key.Raw,
			TopicName:  link.Key.TopicName,
			PageTitle:  link.Key.PageTitle,
			Resolution: apigen.WikilinkPreviewLinkResolution(link.Resolution),
		}
		if link.Page != nil {
			page, err := apipage.ToAPIPage(h.appURL, principal.Space, link.Page, output.Topics[link.Page.TopicID])
			if err != nil {
				return nil, err
			}
			item.Page = &apigen.WikilinkPreviewPage{
				Id:          page.Id,
				Number:      page.Number,
				TopicNumber: page.TopicNumber,
				Title:       page.Title,
				Status:      apigen.WikilinkPreviewPageStatus(page.Status),
				Url:         page.Url,
			}
		}
		links = append(links, item)
	}

	return apigen.CreateWikilinkPreview200JSONResponse{Body: apigen.WikilinkPreview{Links: links}}, nil
}
