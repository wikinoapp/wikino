# typed: strict
# frozen_string_literal: true

class PagePolicy < ApplicationPolicy
  include PolicyConcerns::SpaceContext

  sig { returns(T::Boolean) }
  def show?
    if same_space_member?
      return space_member_record!.active?
    end

    page_record.topic_record!.visibility_public?
  end

  sig { returns(T::Boolean) }
  def update_draft?
    return false unless same_space_member?

    space_member_record!.topic_records.where(id: page_record.topic_id).exists?
  end

  sig { returns(PageRecord) }
  private def page_record
    T.cast(record, PageRecord)
  end
end
