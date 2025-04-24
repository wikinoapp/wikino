# typed: strict
# frozen_string_literal: true

class PagePolicy < ApplicationPolicy
  sig { params(user_record: T.nilable(UserRecord), page_record: PageRecord).void }
  def initialize(user_record:, page_record:)
    @user_record = user_record
    @page_record = page_record
  end

  sig { returns(T.nilable(UserRecord)) }
  attr_reader :user_record
  private :user_record

  sig { returns(PageRecord) }
  attr_reader :page_record
  private :page_record

  sig { returns(T::Boolean) }
  def show?
    if space_member?
      space_member_record!.active?
    else
      page_record.topic_record!.visibility_public?
    end
  end

  sig { returns(T::Boolean) }
  def update_draft?
    return false if space_member_record.nil?

    space_member_record.topic_records.where(id: page_record.topic_id).exists?
  end

  sig { returns(T::Boolean) }
  private def signed_in?
    user_record.present?
  end

  sig { returns(UserRecord) }
  private def user_record!
    user_record.not_nil!
  end

  sig { returns(T.nilable(SpaceMemberRecord)) }
  private def space_member_record
    user_record&.space_member_record(space_record: page_record.space_record!)
  end

  sig { returns(T::Boolean) }
  private def space_member?
    space_member_record.present?
  end

  sig { returns(SpaceMemberRecord) }
  private def space_member_record!
    space_member_record.not_nil!
  end
end
