# typed: strict
# frozen_string_literal: true

class DraftPageRepository < ApplicationRepository
  sig { params(draft_page_record: DraftPageRecord).returns(DraftPage) }
  def to_model(draft_page_record:)
    DraftPage.new(
      database_id: draft_page_record.id,
      modified_at: draft_page_record.modified_at,
      space: SpaceRepository.new.to_model(space_record: draft_page_record.space_record.not_nil!),
      page: PageRepository.new.to_model(page_record: draft_page_record.page_record.not_nil!)
    )
  end
end
