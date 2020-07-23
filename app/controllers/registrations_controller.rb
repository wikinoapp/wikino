# frozen_string_literal: true

class RegistrationsController < Devise::RegistrationsController
  def new
    @new_user = User.new_with_session({}, session)
  end

  def create
    @new_user = User.new(user_params)

    return render(:new) unless @new_user.valid?

    ActiveRecord::Base.transaction do
      @new_user.save!
      team = Team.create!
      team.team_members.create!(user: @new_user)
      team.projects.create!(name: "Personal")
    end

    flash[:notice] = t("messages.sign_up.confirmation_mail_has_sent")
    redirect_to root_path
  end

  private

  def user_params
    params.require(:user).permit(:email, :password)
  end
end
