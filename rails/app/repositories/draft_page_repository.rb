# typed: strict
# frozen_string_literal: true

class DraftPageRepository < ApplicationRepository
  sig { params(draft_page_record: DraftPageRecord).returns(DraftPage) }
  def to_model(draft_page_record:)
    DraftPage.new(
      database_id: draft_page_record.id,
      modified_at: draft_page_record.modified_at,
      space: SpaceRepository.new.to_model(space_record: draft_page_record.space_record.not_nil!),
      page: PageRepository.new.to_model(page_record: draft_page_record.page_record.not_nil!, current_space_member: nil)
    )
  end

  sig do
    params(
      user_record: UserRecord,
      limit: Integer
    ).returns({draft_pages: T::Array[DraftPage], has_more: T::Boolean})
  end
  def find_for_sidebar(user_record:, limit:)
    draft_page_records = DraftPageRecord
      .joins(:space_member_record)
      .where(space_members: {user_id: user_record.id, active: true})
      .preload(:space_record, page_record: [:space_record, topic_record: :space_record])
      .order(modified_at: :desc)
      .limit(limit + 1)

    has_more = draft_page_records.size > limit
    records = has_more ? draft_page_records.first(limit) : draft_page_records.to_a

    draft_pages = records.map do |draft_page_record|
      to_model(draft_page_record:)
    end

    {draft_pages:, has_more:}
  end
end
