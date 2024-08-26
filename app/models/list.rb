# typed: strict
# frozen_string_literal: true

class List < ApplicationRecord
  include Discard::Model

  acts_as_sequenced column: :number, scope: :space_id

  enum :visibility, {
    ListVisibility::Public.serialize => 0,
    ListVisibility::Private.serialize => 1
  }, prefix: true

  belongs_to :space
  has_many :list_members, dependent: :restrict_with_exception
  has_many :notes, dependent: :restrict_with_exception

  scope :public_only, -> { where(visibility: ListVisibility::Public.serialize) }

  sig { params(user: User, role: ListMemberRole, joined_at: ActiveSupport::TimeWithZone).void }
  def add_member(user:, role:, joined_at: Time.current)
    list_members.create!(space: user.space, user:, role: role.serialize, joined_at:)
  end
end
