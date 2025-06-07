# typed: strict
# frozen_string_literal: true

class SpaceMemberRole < T::Enum
  extend T::Sig

  enums do
    Owner = new("owner")
    Member = new("member")
  end

  sig { returns(T::Array[SpaceMemberPermission]) }
  def permissions
    case self
    when Owner
      [
        SpaceMemberPermission::CreateTopic,
        SpaceMemberPermission::CreatePage,
        SpaceMemberPermission::CreateDraftPage,
        SpaceMemberPermission::ExportSpace,
        SpaceMemberPermission::UpdateSpace,
        SpaceMemberPermission::UpdateTopic
      ]
    when Member
      [
        SpaceMemberPermission::CreateTopic,
        SpaceMemberPermission::CreatePage,
        SpaceMemberPermission::CreateDraftPage,
        SpaceMemberPermission::UpdateTopic
      ]
    else
      T.absurd(self)
    end
  end
end
