# typed: strict
# frozen_string_literal: true

class UserRecord < ApplicationRecord
  include Discard::Model

  include ModelConcerns::Viewable

  self.table_name = "users"

  enum :locale, {
    ViewerLocale::En.serialize => 0,
    ViewerLocale::Ja.serialize => 1
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
    class_name: "SpaceRecord",
    through: :active_space_member_records,
    source: :space_record
  has_many :user_session_records, dependent: :restrict_with_exception, foreign_key: :user_id
  has_one :user_password_record, dependent: :restrict_with_exception, foreign_key: :user_id

  sig do
    params(
      email: String,
      atname: String,
      password: String,
      locale: ViewerLocale,
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

  sig { returns(UserEntity) }
  def to_entity
    UserEntity.new(
      database_id: id,
      atname:,
      name:,
      description:,
      time_zone:
    )
  end

  sig { override.returns(T::Boolean) }
  def signed_in?
    true
  end

  sig { override.returns(T.nilable(UserEntity)) }
  def user_entity
    to_entity
  end

  sig { override.params(space: SpaceRecord).returns(ModelConcerns::SpaceViewable) }
  def space_viewer!(space:)
    active_space_member_records.find_by(space_record: space).presence || SpaceVisitor.new(space:)
  end

  sig { override.params(space: SpaceRecord).returns(T::Boolean) }
  def joined_space?(space:)
    active_space_records.where(id: space.id).exists?
  end

  sig { params(topic: TopicRecord).returns(T::Boolean) }
  def joined_topic?(topic:)
    topics.include?(topic)
  end

  sig { params(topic_ids: T::Array[T::Wikino::DatabaseId]).returns(T::Boolean) }
  def joined_all_topics?(topic_ids:)
    joined_topic_ids = topics.pluck(:id)
    topic_ids - joined_topic_ids == []
  end

  sig { override.returns(ViewerLocale) }
  def viewer_locale
    ViewerLocale.deserialize(locale)
  end

  sig { override.returns(TopicRecord::PrivateRelation) }
  def viewable_topics
    TopicRecord
      .left_joins(:member_records)
      .merge(TopicRecord.visibility_public.or(TopicRecord.where(space_record: active_space_records)))
      .distinct
  end

  sig { override.params(topic: TopicRecord).returns(T::Boolean) }
  def can_view_topic?(topic:)
    viewable_topics.where(id: topic.id).exists?
  end

  sig { params(topic: TopicRecord).returns(T::Boolean) }
  def can_update_topic?(topic:)
    topics.where(id: topic.id).exists?
  end

  sig { params(topic: TopicRecord).returns(T::Boolean) }
  def can_destroy_topic?(topic:)
    topic_members.find_by(topic:)&.role_admin? == true
  end

  sig { override.params(page: PageRecord).returns(T::Boolean) }
  def can_trash_page?(page:)
    active_spaces.where(id: page.space_id).exists?
  end

  sig { params(email_confirmation: EmailConfirmationRecord).void }
  def run_after_email_confirmation_success!(email_confirmation:)
    return unless email_confirmation.succeeded?

    if email_confirmation.deserialized_event == EmailConfirmationEvent::EmailUpdate
      update!(email: email_confirmation.email)
    end

    nil
  end
end
