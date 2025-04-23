# typed: strict
# frozen_string_literal: true

class PagePolicy < ApplicationPolicy
  sig { params(space_member_record: T.nilable(SpaceMemberRecord), page_record: PageRecord).void }
  def initialize(space_member_record:, page_record:)
    @space_member_record = space_member_record
    @page_record = page_record
  end

  sig { returns(SpaceMemberRecord) }
  attr_reader :space_member_record
  private :space_member_record

  sig { returns(PageRecord) }
  attr_reader :page_record
  private :page_record

  sig { returns(T::Boolean) }
  def update_draft?
    return false if space_member_record.nil?

    space_member_record.topic_records.where(id: page_record.topic_id).exists?
  end
end
