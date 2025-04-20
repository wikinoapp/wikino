# typed: strict
# frozen_string_literal: true

class PageRepository < ApplicationRepository
  sig { params(page_record: PageRecord).returns(Page) }
  def build_model(page_record:)
    Page.new(
      database_id: page_record.id
    )
  end
end
