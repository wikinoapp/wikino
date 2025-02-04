# typed: strict
# frozen_string_literal: true

module SpaceMemberRole
  class Owner
    extend T::Sig

    sig { returns(String) }
    def self.serialize
      "owner"
    end

    sig { returns(T::Array[SpaceMemberRole::Permission]) }
    def permissions
      [
        SpaceMemberRole::Permission::CreateTopic,
        SpaceMemberRole::Permission::CreatePage,
        SpaceMemberRole::Permission::CreateDraftPage
      ]
    end
  end
end
