# typed: strict
# frozen_string_literal: true

class UserRecord < ApplicationRecord
  include Discard::Model

  self.table_name = "users"

  enum :locale, {
    Locale::En.serialize => 0,
    Locale::Ja.serialize => 1
  }, prefix: true

  has_many :space_member_records,
    dependent: :restrict_with_exception,
    foreign_key: :user_id,
    inverse_of: :user_record
  has_many :active_space_member_records,
    -> { SpaceMemberRecord.active },
    class_name: "SpaceMemberRecord",
    dependent: :restrict_with_exception,
    foreign_key: :user_id,
    inverse_of: :user_record
  has_many :topic_member_records, through: :space_member_records, source: :topic_member_records
  has_many :topic_records, through: :topic_member_records
  has_many :space_records, through: :space_member_records
  has_many :active_space_records,
    -> { SpaceRecord.visible },
    class_name: "SpaceRecord",
    through: :active_space_member_records,
    source: :space_record
  has_many :user_session_records, dependent: :restrict_with_exception, foreign_key: :user_id
  has_one :user_password_record, dependent: :restrict_with_exception, foreign_key: :user_id
  has_one :user_two_factor_auth_record, dependent: :restrict_with_exception, foreign_key: :user_id

  sig do
    params(
      email: String,
      atname: String,
      password: String,
      locale: Locale,
      time_zone: String,
      current_time: ActiveSupport::TimeWithZone
    ).returns(UserRecord)
  end
  def self.create_initial_user!(email:, atname:, password:, locale:, time_zone:, current_time:)
    user = create!(
      email:,
      atname:,
      name: "",
      description: "",
      locale: locale.serialize,
      time_zone:,
      joined_at: current_time
    )
    user.create_user_password_record!(password:)

    user
  end

  sig { returns(T::Boolean) }
  def signed_in?
    true
  end

  sig { params(space_record: SpaceRecord).returns(T.nilable(SpaceMemberRecord)) }
  def space_member_record(space_record:)
    active_space_member_records.find_by(space_record:)
  end

  sig { params(space_record: SpaceRecord).returns(T::Boolean) }
  def joined_space?(space_record:)
    active_space_records.where(id: space_record.id).exists?
  end

  sig { params(topic: TopicRecord).returns(T::Boolean) }
  def joined_topic?(topic:)
    topic_records.include?(topic)
  end

  sig { params(topic_ids: T::Array[T::Wikino::DatabaseId]).returns(T::Boolean) }
  def joined_all_topics?(topic_ids:)
    joined_topic_ids = topic_records.pluck(:id)
    topic_ids - joined_topic_ids == []
  end

  sig { returns(Locale) }
  def viewer_locale
    Locale.deserialize(locale)
  end

  sig { returns(TopicRecord::PrivateRelation) }
  def viewable_topics
    TopicRecord
      .left_joins(:member_records)
      .merge(TopicRecord.visibility_public.or(TopicRecord.where(space_record: active_space_records)))
      .distinct
  end

  sig { params(topic_record: TopicRecord).returns(T::Boolean) }
  def can_destroy_topic?(topic_record:)
    topic_member_records.find_by(topic_record:)&.role_admin? == true
  end

  sig { params(email_confirmation_record: EmailConfirmationRecord).void }
  def run_after_email_confirmation_success!(email_confirmation_record:)
    return unless email_confirmation_record.succeeded?

    if email_confirmation_record.deserialized_event == EmailConfirmationEvent::EmailUpdate
      update!(email: email_confirmation_record.email)
    end

    nil
  end

  sig { returns(T::Boolean) }
  def two_factor_enabled?
    user_two_factor_auth_record&.enabled? || false
  end
end
