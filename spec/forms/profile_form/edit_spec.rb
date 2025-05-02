# typed: false
# frozen_string_literal: true

RSpec.describe ProfileForm::Edit, type: :form do
  it "アットネームが空文字列のとき、エラーになること" do
    form = ProfileForm::Edit.new(atname: "")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Atname can't be blank")
  end

  it "アットネームが `nil` のとき、エラーになること" do
    form = ProfileForm::Edit.new(atname: nil)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Atname can't be blank")
  end

  it "アットネームが21文字のとき、エラーになること" do
    form = ProfileForm::Edit.new(atname: "a" * 21)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Atname is too long (maximum is 20 characters)")
  end

  it "アットネームが予約語のとき、エラーになること" do
    form = ProfileForm::Edit.new(atname: "admin")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Atname cannot be used")
  end

  it "アットネームの形式が不正なとき、エラーになること" do
    form = ProfileForm::Edit.new(atname: "a@b")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Atname is invalid")
  end

  it "アットネームがすでに使われているとき、エラーになること" do
    create(:user_record, atname: "already_used_atname")
    user = create(:user_record)

    form = ProfileForm::Edit.new(user_record: user, atname: "already_used_atname")

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Atname has already been taken")
  end

  it "アットネームが1文字のとき、エラーにならないこと" do
    form = ProfileForm::Edit.new(
      atname: "a",
      **valid_attributes.except(:atname)
    )

    expect(form).to be_valid
  end

  it "アットネームが20文字のとき、エラーにならないこと" do
    form = ProfileForm::Edit.new(
      atname: "a" * 20,
      **valid_attributes.except(:atname)
    )

    expect(form).to be_valid
  end

  it "名前が31文字のとき、エラーになること" do
    form = ProfileForm::Edit.new(name: "a" * 31)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Name is too long (maximum is 30 characters)")
  end

  it "名前が30文字のとき、エラーにならないこと" do
    form = ProfileForm::Edit.new(
      name: "a" * 30,
      **valid_attributes.except(:name)
    )

    expect(form).to be_valid
  end

  it "名前が空文字列のとき、エラーにならないこと" do
    form = ProfileForm::Edit.new(
      name: "",
      **valid_attributes.except(:name)
    )

    expect(form).to be_valid
  end

  it "名前が `nil` のとき、空文字列に変換され、エラーにならないこと" do
    form = ProfileForm::Edit.new(
      name: nil,
      **valid_attributes.except(:name)
    )

    expect(form).to be_valid
    expect(form.name).to eq("")
  end

  it "説明文が151文字のとき、エラーになること" do
    form = ProfileForm::Edit.new(description: "a" * 151)

    expect(form).not_to be_valid
    expect(form.errors.full_messages).to include("Description is too long (maximum is 150 characters)")
  end

  it "説明文が150文字のとき、エラーにならないこと" do
    form = ProfileForm::Edit.new(
      description: "a" * 150,
      **valid_attributes.except(:description)
    )

    expect(form).to be_valid
  end

  it "説明文が空文字列のとき、エラーにならないこと" do
    form = ProfileForm::Edit.new(
      description: "",
      **valid_attributes.except(:description)
    )

    expect(form).to be_valid
  end

  it "説明文が `nil` のとき、空文字列に変換され、エラーにならないこと" do
    form = ProfileForm::Edit.new(
      description: nil,
      **valid_attributes.except(:description)
    )

    expect(form).to be_valid
    expect(form.description).to eq("")
  end

  private def valid_attributes
    user_record = create(:user_record)

    {
      user_record:,
      atname: "a",
      name: "a",
      description: "a"
    }
  end
end
