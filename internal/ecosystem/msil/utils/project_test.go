package utils

import (
	"cli/internal/common"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/beevik/etree"
)

var legacyProjectFileData = `<?xml version="1.0" encoding="utf-8"?>
<Project ToolsVersion="14.0" DefaultTargets="Build" xmlns="http://schemas.microsoft.com/developer/msbuild/2003">
  <Import Project="$(MSBuildExtensionsPath)\$(MSBuildToolsVersion)\Microsoft.Common.props" Condition="Exists('$(MSBuildExtensionsPath)\$(MSBuildToolsVersion)\Microsoft.Common.props')" />
  <PropertyGroup>
    <Configuration Condition=" '$(Configuration)' == '' ">Debug</Configuration>
    <Platform Condition=" '$(Platform)' == '' ">AnyCPU</Platform>
    <ProjectGuid>{7DA69E3D-DE6C-4F1F-B1BF-841E88A9B1E8}</ProjectGuid>
    <OutputType>Exe</OutputType>
    <AppDesignerFolder>Properties</AppDesignerFolder>
    <RootNamespace>ConsoleApplication1</RootNamespace>
    <AssemblyName>ConsoleApplication1</AssemblyName>
    <TargetFrameworkVersion>v4.5.2</TargetFrameworkVersion>
    <FileAlignment>512</FileAlignment>
    <AutoGenerateBindingRedirects>true</AutoGenerateBindingRedirects>
  </PropertyGroup>
  <PropertyGroup Condition=" '$(Configuration)|$(Platform)' == 'Debug|AnyCPU' ">
    <PlatformTarget>AnyCPU</PlatformTarget>
    <DebugSymbols>true</DebugSymbols>
    <DebugType>full</DebugType>
    <Optimize>false</Optimize>
    <OutputPath>bin\Debug\</OutputPath>
    <DefineConstants>DEBUG;TRACE</DefineConstants>
    <ErrorReport>prompt</ErrorReport>
    <WarningLevel>4</WarningLevel>
  </PropertyGroup>
  <PropertyGroup Condition=" '$(Configuration)|$(Platform)' == 'Release|AnyCPU' ">
    <PlatformTarget>AnyCPU</PlatformTarget>
    <DebugType>pdbonly</DebugType>
    <Optimize>true</Optimize>
    <OutputPath>bin\Release\</OutputPath>
    <DefineConstants>TRACE</DefineConstants>
    <ErrorReport>prompt</ErrorReport>
    <WarningLevel>4</WarningLevel>
  </PropertyGroup>
  <ItemGroup>
    <Reference Include="Newtonsoft.Json, Version=13.0.0.0, Culture=neutral, PublicKeyToken=30ad4fe6b2a6aeed, processorArchitecture=MSIL">
      <HintPath>..\lib\Newtonsoft.Json.13.0.+sp1\lib\net45\Newtonsoft.Json.dll</HintPath>
      <Private>True</Private>
    </Reference>
    <Reference Include="System" />
    <Reference Include="System.Core" />
    <Reference Include="System.Xml.Linq" />
    <Reference Include="System.Data.DataSetExtensions" />
    <Reference Include="Microsoft.CSharp" />
    <Reference Include="System.Data" />
    <Reference Include="System.Net.Http" />
    <Reference Include="System.Xml" />
  </ItemGroup>
  <ItemGroup>
    <Compile Include="Program.cs" />
    <Compile Include="Properties\AssemblyInfo.cs" />
  </ItemGroup>
  <ItemGroup>
    <None Include="App.config" />
  </ItemGroup>
  <Import Project="$(MSBuildToolsPath)\Microsoft.CSharp.targets" />
  <!-- To modify your build process, add your task inside one of the targets below and uncomment it. 
       Other similar extension points exist, see Microsoft.Common.targets.
  <Target Name="BeforeBuild">
  </Target>
  <Target Name="AfterBuild">
  </Target>
  -->
</Project>`

func TestBadAttrs(t *testing.T) {
	data := `<Project bbbb="Microsoft.NET.Sdk">

	  <PropertyGroup>
	    <TargetFramework>netstandard2.0</TargetFramework>
	    <Authors>authorname</Authors>
	    <PackageId>mypackageid</PackageId>
	    <Company>mycompanyname</Company>
	  </PropertyGroup>

	</Project>`

	doc := etree.NewDocument()
	_, _ = doc.ReadFrom(strings.NewReader(data))
	format, err := inspectProjectFormat(doc)
	if err == nil {
		t.Fatalf("should return err")
	}

	if format != FormatUnknown {
		t.Fatalf("bad format, expected unkown, got %d", format)
	}
}

func TestBadElement(t *testing.T) {
	data := `<NotProject Sdk="Microsoft.NET.Sdk">

	  <PropertyGroup>
	    <TargetFramework>netstandard2.0</TargetFramework>
	    <Authors>authorname</Authors>
	    <PackageId>mypackageid</PackageId>
	    <Company>mycompanyname</Company>
	  </PropertyGroup>

	</NotProject>`

	doc := etree.NewDocument()
	_, _ = doc.ReadFrom(strings.NewReader(data))
	format, err := inspectProjectFormat(doc)
	if err == nil {
		t.Fatalf("should return err")
	}

	if format != FormatUnknown {
		t.Fatalf("bad format, expected unkown, got %d", format)
	}
}

func TestSdkFormat(t *testing.T) {
	// test data taken from https://learn.microsoft.com/en-us/nuget/resources/check-project-format
	// then added deps manually
	data := `<Project Sdk="Microsoft.NET.Sdk">

	  <PropertyGroup>
	    <TargetFramework>netstandard2.0</TargetFramework>
	    <Authors>authorname</Authors>
	    <PackageId>mypackageid</PackageId>
	    <Company>mycompanyname</Company>
	  </PropertyGroup>

    <ItemGroup>
      <PackageReference Include="Newtonsoft.Json">
        <Version>13.0.3</Version>
      </PackageReference>
      <PackageReference Include="WebActivatorEx">
        <Version>2.2.0</Version>
      </PackageReference>
  </ItemGroup>
	</Project>`

	doc := etree.NewDocument()
	_, _ = doc.ReadFrom(strings.NewReader(data))
	format, err := inspectProjectFormat(doc)
	if err != nil {
		t.Fatalf("had error %v", err)
	}

	if format != FormatSdk {
		t.Fatalf("bad format, expected %d, got %d", FormatSdk, format)
	}
}

func TestPackageReferenceCommentedOut(t *testing.T) {
	data := `<Project ToolsVersion="14.0" DefaultTargets="Build" xmlns="http://schemas.microsoft.com/developer/msbuild/2003">
  <!--
    <ItemGroup>
      <PackageReference Include="Newtonsoft.Json">
        <Version>13.0.3</Version>
      </PackageReference>
      <PackageReference Include="WebActivatorEx">
        <Version>2.2.0</Version>
      </PackageReference>
  </ItemGroup>
  -->
	</Project>`

	doc := etree.NewDocument()
	_, _ = doc.ReadFrom(strings.NewReader(data))
	format, err := inspectProjectFormat(doc)
	if err != nil {
		t.Fatalf("had error %v", err)
	}

	// since the PackageReference are commented out, we should not detect it as Migrated, but instead as legacy
	if format != FormatLegacy {
		t.Fatalf("bad format, expected %d, got %d", FormatLegacy, format)
	}

}

func TestFormatLegacy(t *testing.T) {

	doc := etree.NewDocument()
	_, _ = doc.ReadFrom(strings.NewReader(legacyProjectFileData))
	format, err := inspectProjectFormat(doc)
	if err != nil {
		t.Fatalf("had error %v", err)
	}

	if format != FormatLegacy {
		t.Fatalf("bad format, expected %d, got %d", FormatLegacy, format)
	}
}

func TestMigratedFormat(t *testing.T) {
	data := `<?xml version="1.0" encoding="utf-8"?>
<Project ToolsVersion="14.0" DefaultTargets="Build" xmlns="http://schemas.microsoft.com/developer/msbuild/2003">
  <Import Project="$(MSBuildExtensionsPath)\$(MSBuildToolsVersion)\Microsoft.Common.props" Condition="Exists('$(MSBuildExtensionsPath)\$(MSBuildToolsVersion)\Microsoft.Common.props')" />
  <PropertyGroup>
    <Configuration Condition=" '$(Configuration)' == '' ">Debug</Configuration>
    <Platform Condition=" '$(Platform)' == '' ">AnyCPU</Platform>
    <ProjectGuid>{7DA69E3D-DE6C-4F1F-B1BF-841E88A9B1E8}</ProjectGuid>
    <OutputType>Exe</OutputType>
    <AppDesignerFolder>Properties</AppDesignerFolder>
    <RootNamespace>ConsoleApplication1</RootNamespace>
    <AssemblyName>ConsoleApplication1</AssemblyName>
    <TargetFrameworkVersion>v4.5.2</TargetFrameworkVersion>
    <FileAlignment>512</FileAlignment>
    <AutoGenerateBindingRedirects>true</AutoGenerateBindingRedirects>
  </PropertyGroup>
  <PropertyGroup Condition=" '$(Configuration)|$(Platform)' == 'Debug|AnyCPU' ">
    <PlatformTarget>AnyCPU</PlatformTarget>
    <DebugSymbols>true</DebugSymbols>
    <DebugType>full</DebugType>
    <Optimize>false</Optimize>
    <OutputPath>bin\Debug\</OutputPath>
    <DefineConstants>DEBUG;TRACE</DefineConstants>
    <ErrorReport>prompt</ErrorReport>
    <WarningLevel>4</WarningLevel>
  </PropertyGroup>
  <PropertyGroup Condition=" '$(Configuration)|$(Platform)' == 'Release|AnyCPU' ">
    <PlatformTarget>AnyCPU</PlatformTarget>
    <DebugType>pdbonly</DebugType>
    <Optimize>true</Optimize>
    <OutputPath>bin\Release\</OutputPath>
    <DefineConstants>TRACE</DefineConstants>
    <ErrorReport>prompt</ErrorReport>
    <WarningLevel>4</WarningLevel>
  </PropertyGroup>
  <ItemGroup>
    <Reference Include="System" />
    <Reference Include="System.Core" />
    <Reference Include="System.Xml.Linq" />
    <Reference Include="System.Data.DataSetExtensions" />
    <Reference Include="Microsoft.CSharp" />
    <Reference Include="System.Data" />
    <Reference Include="System.Net.Http" />
    <Reference Include="System.Xml" />
  </ItemGroup>
  <ItemGroup>
    <Compile Include="Program.cs" />
    <Compile Include="Properties\AssemblyInfo.cs" />
  </ItemGroup>
  <ItemGroup>
    <None Include="App.config" />
  </ItemGroup>
  <ItemGroup>
    <PackageReference Include="Newtonsoft.Json">
      <Version>13.0.3</Version>
    </PackageReference>
    <PackageReference Include="WebActivatorEx">
      <Version>2.2.0</Version>
    </PackageReference>
  </ItemGroup>
  <Import Project="$(MSBuildToolsPath)\Microsoft.CSharp.targets" />
  <!-- To modify your build process, add your task inside one of the targets below and uncomment it. 
       Other similar extension points exist, see Microsoft.Common.targets.
  <Target Name="BeforeBuild">
  </Target>
  <Target Name="AfterBuild">
  </Target>
  -->
</Project>`

	doc := etree.NewDocument()
	_, _ = doc.ReadFrom(strings.NewReader(data))
	format, err := inspectProjectFormat(doc)
	if err != nil {
		t.Fatalf("had error %v", err)
	}

	if format != FormatMigrated {
		t.Fatalf("bad format, expected %d, got %d", FormatMigrated, format)
	}
}

func TestDetectProjectFormatUnknownLegacy(t *testing.T) {

	root, _ := os.MkdirTemp("", "test_seal_cli_*")

	projPath := filepath.Join(root, "test.csproj")
	_ = common.DumpBytes(projPath, []byte(legacyProjectFileData))

	f, err := DetectProjectFormat(projPath)
	if err != nil {
		t.Fatalf("err %v", err)
	}

	if f != FormatUnknown {
		t.Fatalf("bad format %v", f)
	}
}

func TestDetectProjectFormatLegacyPackagesConfig(t *testing.T) {

	root, _ := os.MkdirTemp("", "test_seal_cli_*")

	projPath := filepath.Join(root, "test.csproj")
	packagesConfigPath := filepath.Join(root, "packages.config")
	_ = common.DumpBytes(projPath, []byte(legacyProjectFileData))
	_ = common.DumpBytes(packagesConfigPath, []byte(""))

	f, err := DetectProjectFormat(projPath)
	if err != nil {
		t.Fatalf("err %v", err)
	}

	if f != FormatLegacyPackagesConfig {
		t.Fatalf("bad format %v", f)
	}
}

func TestDetectProjectFormatLegacyProjectJson(t *testing.T) {

	root, _ := os.MkdirTemp("", "test_seal_cli_*")

	projPath := filepath.Join(root, "test.csproj")
	packagesConfigPath := filepath.Join(root, "project.json")
	_ = common.DumpBytes(projPath, []byte(legacyProjectFileData))
	_ = common.DumpBytes(packagesConfigPath, []byte(""))

	f, err := DetectProjectFormat(projPath)
	if err != nil {
		t.Fatalf("err %v", err)
	}

	if f != FormatLegacyProjectJson {
		t.Fatalf("bad format %v", f)
	}
}
