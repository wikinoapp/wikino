# typed: strict
# frozen_string_literal: true

class TopicMemberRecord < ApplicationRecord
  self.table_name = "topic_members"

  belongs_to :space_record, foreign_key: :space_id
  belongs_to :topic_record, foreign_key: :topic_id
  belongs_to :space_member_record, foreign_key: :space_member_id

  sig { params(time: ActiveSupport::TimeWithZone).void }
  def update_last_page_modified_at!(time:)
    update!(last_page_modified_at: time)
  end
end
