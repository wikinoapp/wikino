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
  has_many :memberships, class_name: "ListMembership", dependent: :restrict_with_exception
  has_many :notes, dependent: :restrict_with_exception

  scope :public_only, -> { where(visibility: ListVisibility::Public.serialize) }

  sig { params(member: User, role: ListMemberRole, joined_at: ActiveSupport::TimeWithZone).void }
  def add_member!(member:, role:, joined_at: Time.current)
    memberships.create!(space: member.space, member:, role: role.serialize, joined_at:)

    nil
  end
end
