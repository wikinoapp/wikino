# typed: strict
# frozen_string_literal: true

class DraftPageRepository < ApplicationRepository
  include RepositoryConcerns::Pageable

  sig { params(draft_page_record: DraftPageRecord).returns(DraftPage) }
  def to_model(draft_page_record:)
    DraftPage.new(
      database_id: draft_page_record.id,
      modified_at: draft_page_record.modified_at
    )
  end
end
