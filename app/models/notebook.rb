# typed: strict
# frozen_string_literal: true

class Notebook < ApplicationRecord
  include Discard::Model

  acts_as_sequenced column: :number, scope: :space_id

  enum :visibility, {
    NotebookVisibility::Public.serialize => 0,
    NotebookVisibility::Private.serialize => 1
  }, prefix: true

  belongs_to :space
  has_many :memberships, class_name: "NotebookMembership", dependent: :restrict_with_exception
  has_many :notes, dependent: :restrict_with_exception

  scope :public_only, -> { where(visibility: NotebookVisibility::Public.serialize) }

  sig { params(member: User, role: NotebookMemberRole, joined_at: ActiveSupport::TimeWithZone).void }
  def add_member!(member:, role:, joined_at: Time.zone.now)
    memberships.create!(space: member.space, member:, role: role.serialize, joined_at:)

    nil
  end
end
